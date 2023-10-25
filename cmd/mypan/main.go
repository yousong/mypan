// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"mypan/pkg/client"
	"mypan/pkg/config"
	"mypan/pkg/store"
	"mypan/pkg/util"

	"github.com/golang/glog"
	"github.com/jedib0t/go-pretty/progress"
	ptable "github.com/jedib0t/go-pretty/table"
	ptext "github.com/jedib0t/go-pretty/text"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var unixTimeFormatter = ptext.NewUnixTimeTransformer(time.RFC3339, time.Local)

type Render struct {
	format string
}

func NewRender(format string) Render {
	rdr := Render{
		format: format,
	}
	return rdr
}

func (rdr Render) pRender(w ptable.Writer) {
	w.SortBy([]ptable.SortBy{
		{Number: 1, Mode: ptable.Asc},
	})
	w.Style().Options.DrawBorder = false
	w.Style().Options.SeparateRows = false
	w.Style().Options.SeparateColumns = false
	out := w.Render()
	fmt.Printf("%s\n", out)
}

func (rdr Render) Render(v interface{}) {
	if rdr.format == "json" {
		rdr.RenderAsJSON(v)
		return
	}
	switch val := v.(type) {
	case client.ListResponse:
		rdr.RenderListResponse(val)
	case client.ListAllResponse:
		rdr.RenderListAllResponse(val)
	default:
		rdr.RenderAsJSON(v)
	}
}

func (rdr Render) RenderAsJSON(v interface{}) {
	fmt.Printf("%s\n", util.MustMarshalJSON(v))
}

func (rdr Render) RenderListResponse(resp client.ListResponse) {
	w := ptable.NewWriter()
	for _, f := range resp.List {
		name := f.ServerFilename
		var sizeCol string
		if f.IsDir == 0 {
			sizeCol = strconv.FormatUint(f.Size, 10)
		} else {
			name += "/"
			if f.Empty != nil && *f.Empty == 0 {
				sizeCol = "*"
			} else {
				sizeCol = ""
			}
		}
		mtimeStr := unixTimeFormatter(int64(f.LocalMtime))
		w.AppendRow([]interface{}{
			name,
			sizeCol,
			mtimeStr,
			f.Md5,
		})
	}
	rdr.pRender(w)
}

func (rdr Render) RenderListAllResponse(resp client.ListAllResponse) {
	w := ptable.NewWriter()
	for _, f := range resp.List {
		name := f.Path
		var sizeCol string
		if f.IsDir == 0 {
			sizeCol = strconv.FormatUint(f.Size, 10)
		} else {
			name += "/"
			sizeCol = "-"
		}
		mtimeStr := unixTimeFormatter(int64(f.LocalMtime))
		w.AppendRow([]interface{}{
			name,
			sizeCol,
			mtimeStr,
			f.Md5,
		})
	}
	rdr.pRender(w)
}

type MyApp struct {
	ctx     context.Context
	timeout time.Duration
	render  Render

	dstClient client.ClientI
	dirStore  store.StoreI
	jsonStore store.StoreSerdeI

	progress *progress.Progress
}

func NewMyApp() MyApp {
	myApp := MyApp{
		ctx:    context.Background(),
		render: NewRender("json"),
	}
	return myApp
}

func (myApp MyApp) syncAction(cCtx *cli.Context, src, dst string, up bool) error {
	if src == "" || dst == "" {
		return cli.Exit("src and dst arguments are required", 1)
	}
	var (
		ctx       = myApp.ctx
		dstClient = myApp.dstClient
		jsonStore = myApp.jsonStore
	)

	var opts []SyncOpt
	if cCtx.Bool("dryrun") {
		opts = append(opts, DryRun(true))
	}
	if cCtx.Bool("nodelete") {
		opts = append(opts, NoDelete(true))
	}
	if progress := myApp.progress; progress != nil {
		opts = append(opts, Progress(progress))
	}
	dstCacheStore, err := store.NewFileCacheStore(
		config.StoreKeyDstCacheEntry,
		jsonStore,
		NewDstCacheEntry,
	)
	if err != nil {
		return cli.Exit(errors.Wrap(err, "dst cache store"), 1)
	}
	srcCacheStore, err := store.NewFileCacheStore(
		config.StoreKeySrcCacheEntry,
		jsonStore,
		NewSrcCacheEntry,
	)
	if err != nil {
		return cli.Exit(errors.Wrap(err, "src cache store"), 1)
	}
	var su *Sync
	if up {
		su = NewSyncUp(src, dst, dstClient, srcCacheStore, dstCacheStore, opts...)
	} else {
		opts = append(opts, Continue())
		su = NewSyncDown(src, dst, dstClient, srcCacheStore, dstCacheStore, opts...)
	}
	myApp.progressRender()
	if err := su.Do(ctx); err != nil {
		return err
	}
	return nil
}

func (myApp MyApp) copyMoveAction(
	cCtx *cli.Context,
	action func(context.Context, string, string) (client.FileManagerResponse, error),
) error {
	remotepath0 := cCtx.Args().Get(0)
	remotepath1 := cCtx.Args().Get(1)
	if remotepath0 == "" || remotepath1 == "" {
		return cli.Exit("remotepath0 and remotepath1 arguments are required", 1)
	}
	resp, err := action(myApp.ctx, remotepath0, remotepath1)
	if err != nil {
		return cli.Exit(err, 1)
	}
	myApp.render.Render(resp)
	return nil
}

func (myApp MyApp) Run(args []string) {
	cfg := config.Global

	var ()
	app := &cli.App{
		Name:          "mypan",
		Usage:         "A baidu netdisk client",
		AllowExtFlags: true,
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "appid", Value: cfg.AppID, Destination: &cfg.AppID, EnvVars: []string{"MYPAN_APPID"}},
			&cli.StringFlag{Name: "appkey", Value: cfg.AppKey, Destination: &cfg.AppKey, EnvVars: []string{"MYPAN_APPKEY"}},
			&cli.StringFlag{Name: "secretkey", Value: cfg.SecretKey, Destination: &cfg.SecretKey, EnvVars: []string{"MYPAN_SECRETKEY"}},
			&cli.StringFlag{Name: "appbasedir", Value: cfg.AppBaseDir, Destination: &cfg.AppBaseDir, EnvVars: []string{"MYPAN_APPBASEDIR"}},
			&cli.PathFlag{Name: "rundir", Value: cfg.RunDir, Destination: &cfg.RunDir, EnvVars: []string{"MYPAN_RUNDIR"}},

			&cli.DurationFlag{Name: "timeout", Destination: &myApp.timeout},
			&cli.BoolFlag{Name: "noprogress"},
			&cli.StringFlag{
				Name:  "format",
				Value: "json",
				Usage: "allowed values are json, table",
				Action: func(cCtx *cli.Context, v string) error {
					if v != "json" && v != "table" {
						return fmt.Errorf("invalid format %q, allowed values are json, table", v)
					}
					myApp.render = NewRender(v)
					return nil
				}},
		},
		Before: func(cCtx *cli.Context) error {
			// progress
			if !cCtx.Bool("noprogress") {
				progress := &progress.Progress{}
				progress.Style().Options.TimeInProgressPrecision = time.Second
				myApp.progress = progress
			}
			// ctx
			if timeout := myApp.timeout; timeout != 0 {
				myApp.ctx, _ = context.WithTimeout(myApp.ctx, timeout)
			}
			myApp.ctx, _ = signal.NotifyContext(myApp.ctx, syscall.SIGINT, syscall.SIGTERM)

			// dir store
			var err error
			myApp.dirStore, err = store.NewDirStore(cfg.RunDir)
			if err != nil {
				return errors.Wrap(err, "new dir store")
			}
			// dst client
			var accessAuth client.AccessAuth
			myApp.jsonStore = store.NewJSONStore(myApp.dirStore)
			if err := myApp.jsonStore.Get(config.StoreKeyAccessAuth, &accessAuth); err != nil {
				glog.Warningf("load access auth: %v", err)
			}
			clientCfg := client.Config{
				AppID:      cfg.AppID,
				AppKey:     cfg.AppKey,
				SecretKey:  cfg.SecretKey,
				AppBaseDir: cfg.AppBaseDir,

				AccessAuth: accessAuth,
			}
			myApp.dstClient = client.New(clientCfg)
			return nil
		},
		ExitErrHandler: func(cCtx *cli.Context, err error) {
			if err == nil {
				return
			}
			if _, ok := err.(cli.ExitCoder); ok {
				cli.HandleExitCoder(err)
			} else if _, ok := err.(cli.MultiError); ok {
				cli.HandleExitCoder(err)
			} else {
				glog.Errorf("%v", err)
				cli.OsExiter(1)
			}
		},
		Commands: []*cli.Command{
			{
				Name: "auth",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "refresh"},
				},
				Action: func(cCtx *cli.Context) error {
					authMan := NewAuthMan(myApp.dstClient, myApp.jsonStore)
					if err := authMan.Auth(myApp.ctx); err != nil {
						return cli.Exit(err, 1)
					}
					if cCtx.Bool("refresh") {
						err := authMan.Refresh(myApp.ctx)
						if err != nil {
							return cli.Exit(err, 1)
						}
					}
					return nil
				},
			},
			{
				Name: "quota",
				Action: func(cCtx *cli.Context) error {
					resp, err := myApp.dstClient.Quota(myApp.ctx)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name: "uinfo",
				Action: func(cCtx *cli.Context) error {
					resp, err := myApp.dstClient.UInfo(myApp.ctx)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name:      "ls",
				ArgsUsage: "remotepath",
				Aliases:   []string{"list"},
				Action: func(cCtx *cli.Context) error {
					dir := cCtx.Args().First()
					resp, err := myApp.dstClient.ListEx(myApp.ctx, dir)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name:      "lsa",
				Aliases:   []string{"listall"},
				ArgsUsage: "remotepath",
				Action: func(cCtx *cli.Context) error {
					dir := cCtx.Args().First()
					resp, err := myApp.dstClient.ListAllEx(myApp.ctx, dir)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name:      "stat",
				ArgsUsage: "[remotepath]",
				Action: func(cCtx *cli.Context) error {
					relpaths := cCtx.Args().Slice()
					resp, err := myApp.dstClient.FileMetasByPath(myApp.ctx, relpaths)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp.List)
					return nil
				},
			},
			{
				Name:      "rm",
				Aliases:   []string{"remove"},
				ArgsUsage: "remotepath...",
				Action: func(cCtx *cli.Context) error {
					filelist := cCtx.Args().Slice()
					if len(filelist) == 0 {
						return cli.Exit("filelist argument is required", 1)
					}
					resp, err := myApp.dstClient.DeleteMulti(myApp.ctx, filelist)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name:      "up",
				Aliases:   []string{"upload"},
				ArgsUsage: "localpath remotepath",
				Action: func(cCtx *cli.Context) error {
					src := cCtx.Args().Get(0)
					dst := cCtx.Args().Get(1)
					if src == "" || dst == "" {
						return cli.Exit("src and dst arguments are required", 1)
					}
					myApp.progressRender()
					resp, err := myApp.dstClient.Upload(myApp.ctx, src, dst)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name: "syncup",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dryrun"},
					&cli.BoolFlag{Name: "nodelete"},
				},
				ArgsUsage: "localpath remotepath",
				Action: func(cCtx *cli.Context) error {
					src := cCtx.Args().Get(0)
					dst := cCtx.Args().Get(1)
					return myApp.syncAction(cCtx, src, dst, true)
				},
			},
			{
				Name:    "down",
				Aliases: []string{"download"},
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "continue", Aliases: []string{"c"}},
				},
				ArgsUsage: "remotepath localpath",
				Action: func(cCtx *cli.Context) error {
					relpath := cCtx.Args().Get(0)
					outpath := cCtx.Args().Get(1)
					myApp.progressRender()
					downMan := NewDownMan(myApp.dstClient).
						Continue(cCtx.Bool("continue")).
						Progress(myApp.progress)
					err := downMan.Down(myApp.ctx, relpath, outpath)
					if err != nil {
						return cli.Exit(err, 1)
					}
					return nil
				},
			},
			{
				Name: "syncdown",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dryrun"},
					&cli.BoolFlag{Name: "nodelete"},
					&cli.BoolFlag{Name: "continue", Aliases: []string{"c"}},
				},
				ArgsUsage: "remotepath localpath",
				Action: func(cCtx *cli.Context) error {
					dst := cCtx.Args().Get(0)
					src := cCtx.Args().Get(1)
					return myApp.syncAction(cCtx, src, dst, false)
				},
			},
			{
				Name:      "rename",
				ArgsUsage: "remotepath newname",
				Action: func(cCtx *cli.Context) error {
					path := cCtx.Args().Get(0)
					newname := cCtx.Args().Get(1)
					if path == "" || newname == "" {
						return cli.Exit("path and newname arguments are required", 1)
					}
					resp, err := myApp.dstClient.Rename(myApp.ctx, path, newname)
					if err != nil {
						return cli.Exit(err, 1)
					}
					myApp.render.Render(resp)
					return nil
				},
			},
			{
				Name:      "mv",
				Aliases:   []string{"move"},
				ArgsUsage: "remotepath0 remotepath1",
				Action: func(cCtx *cli.Context) error {
					return myApp.copyMoveAction(cCtx, myApp.dstClient.Move)
				},
			},
			{
				Name:      "cp",
				Aliases:   []string{"copy"},
				ArgsUsage: "remotepath0 remotepath1",
				Action: func(cCtx *cli.Context) error {
					return myApp.copyMoveAction(cCtx, myApp.dstClient.Copy)
				},
			},
			{
				Name: "version",
				Action: func(cCtx *cli.Context) error {
					version := util.VersionFromBuildInfo()
					fmt.Println(version)
					return nil
				},
			},
		},
		Authors: []*cli.Author{
			&cli.Author{Name: "Yousong Zhou", Email: "yszhou4tech@gmail.com"},
		},
	}
	app.Run(args)
	myApp.progreseStop()
}

func (myApp MyApp) progressRender() {
	if progress := myApp.progress; progress != nil {
		go progress.Render()
	}
}

func (myApp MyApp) progreseStop() {
	if p := myApp.progress; p != nil {
		select {
		case <-myApp.ctx.Done():
		case <-time.After(progress.DefaultUpdateFrequency):
		}
		p.Stop()
	}
}

func main() {
	flag.Lookup("logtostderr").Value.Set("true")
	NewMyApp().Run(os.Args)
}
