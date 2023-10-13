package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"syscall"
	"time"

	"mypan/pkg/client"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

const (
	defaultReaddirTTL = 30 * time.Second
	defaultEntryTTL   = 30 * time.Second
	defaultAttrTTL    = 30 * time.Second
)

type MypanNode struct {
	fs.Inode

	Client client.ClientI
	Path   string
	FsId   uint64

	// TODO mu
	listResp   client.ListResponse
	listRespTs time.Time

	metaResp   client.FileMetaResponse
	metaRespTs time.Time
	attr       fuse.Attr
	attrTs     time.Time
}

func Mount(
	client client.ClientI,
	path string,
	fsId uint64,
	targetDir string,
	mntOpts []string,
) (*fuse.Server, error) {
	mn := &MypanNode{
		Client: client,
		Path:   path,
		FsId:   fsId,
	}
	fsOpts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug:       true,
			DirectMount: true,
			Options:     mntOpts,
		},
	}
	return fs.Mount(targetDir, mn, fsOpts)
}

var _ = (fs.InodeEmbedder)((*MypanNode)(nil))

func (mn *MypanNode) list(ctx context.Context) (client.ListResponse, syscall.Errno) {
	if time.Now().Add(-defaultReaddirTTL).Before(mn.listRespTs) {
		return mn.listResp, 0
	}
	resp, err := mn.Client.ListEx(ctx, mn.Path)
	if err != nil {
		return resp, syscall.EIO
	}
	mn.listResp = resp
	mn.listRespTs = time.Now()
	return resp, 0
}

func (mn *MypanNode) meta(ctx context.Context) (client.FileMetaResponse, syscall.Errno) {
	// TODO TTL should not be longer than metaResp.DLink expire time
	if time.Now().Add(-defaultAttrTTL).Before(mn.metaRespTs) {
		return mn.metaResp, 0
	}
	resp, err := mn.Client.FileMeta(ctx, mn.FsId)
	if err != nil {
		return resp, syscall.EIO
	}
	mn.metaResp = resp
	mn.metaRespTs = time.Now()
	return resp, 0
}

func isDirToMode(isDir int) uint32 {
	if isDir != 0 {
		return syscall.S_IFDIR
	} else {
		return syscall.S_IFREG
	}
}

var _ = (fs.NodeReaddirer)((*MypanNode)(nil))

func (mn *MypanNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	resp, errno := mn.list(ctx)
	if errno != 0 {
		return nil, errno
	}
	dirEnts := make([]fuse.DirEntry, 0, len(resp.List))
	for _, ent := range resp.List {
		dirEnt := fuse.DirEntry{
			Mode: isDirToMode(ent.IsDir),
			Name: ent.ServerFilename,
			Ino:  ent.FsId,
		}
		dirEnts = append(dirEnts, dirEnt)
	}
	dirStream := fs.NewListDirStream(dirEnts)
	return dirStream, 0
}

var _ = (fs.NodeLookuper)((*MypanNode)(nil))

func (mn *MypanNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	resp, errno := mn.list(ctx)
	if errno != 0 {
		return nil, errno
	}
	for _, ent := range resp.List {
		if ent.ServerFilename != name {
			continue
		}
		mode := isDirToMode(ent.IsDir)
		stableAttr := fs.StableAttr{
			Mode: mode,
			Ino:  ent.FsId,
		}
		attr := fuse.Attr{
			Ino:   ent.FsId,
			Size:  ent.Size,
			Mtime: ent.LocalMtime,
			Ctime: ent.LocalCtime,
			Mode:  mode,
		}
		mn1 := &MypanNode{
			Client: mn.Client,
			Path:   ent.Path,
			FsId:   ent.FsId,

			attr:   attr,
			attrTs: time.Now(),
		}

		out.EntryValid = uint64(defaultEntryTTL.Seconds())
		out.AttrValid = uint64(defaultAttrTTL.Seconds())
		out.Attr = attr
		child := mn.NewInode(ctx, mn1, stableAttr)
		return child, 0
	}
	return nil, syscall.ENOENT
}

var _ = (fs.NodeGetattrer)((*MypanNode)(nil))

func (mn *MypanNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if now := time.Now(); now.Add(-defaultAttrTTL).Before(mn.attrTs) {
		out.Attr = mn.attr
		out.AttrValid = uint64(now.Sub(mn.attrTs).Seconds())
	}

	metaResp, errno := mn.meta(ctx)
	if errno != 0 {
		return errno
	}
	attr := fuse.Attr{
		Ino:   metaResp.FsId,
		Size:  metaResp.Size,
		Mtime: metaResp.LocalMtime,
		Ctime: metaResp.LocalCtime,
		Mode:  isDirToMode(metaResp.IsDir),
	}
	mn.attr = attr
	mn.attrTs = time.Now()
	out.Attr = mn.attr
	out.AttrValid = uint64(defaultAttrTTL.Seconds())
	return 0
}

var _ = (fs.NodeOpener)((*MypanNode)(nil))

func (mn *MypanNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// TODO
	//return nil, fuse.FOPEN_KEEP_CACHE, 0
	return nil, 0, 0
}

var _ = (fs.NodeReader)((*MypanNode)(nil))

func (mn *MypanNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	metaResp, errno := mn.meta(ctx)
	if errno != 0 {
		return nil, errno
	}
	resp, err := mn.Client.DownloadByDLink(ctx, metaResp.DLink, func(req *http.Request) {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", off))
	})
	if err != nil {
		return nil, syscall.EIO
	}
	defer resp.Body.Close()

	var (
		readn int
		res   fuse.ReadResult
	)
	// The loop is ugly.  It seems the kernel will think it's the end and
	// next read will return EOF if we do not fill full dest buffer.
	for {
		n, err := resp.Body.Read(dest[readn:])
		readn += n
		if err != nil {
			if err != io.EOF {
				errno = syscall.EIO
			}
			break
		}
		if readn >= len(dest) {
			break
		}
		select {
		case <-ctx.Done():
			break
		default:
		}
	}
	if readn > 0 {
		res = fuse.ReadResultData(dest[:readn])
	}
	return res, errno
}

// TODO
// - readdir, lookup都需要吗
// - stat
