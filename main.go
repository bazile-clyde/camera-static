package main

import (
	"context"
	"fmt"
	vidio "github.com/AlexEidt/Vidio"
	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/gostream/codec/x264"
	gimage "go.viam.com/rdk/gostream/image"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"image"
)

var model = resource.NewModel("viam", "camera", "static")

func init() {
	resource.RegisterComponent(camera.API, model, resource.Registration[camera.Camera, resource.NoNativeConfig]{
		Constructor: newCamera,
	})
}

func main() {
	ctx := context.Background()
	logger := logging.NewLogger("static_camera")

	mod, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		panic(err)
	}

	if err = mod.AddModelFromRegistry(ctx, camera.API, model); err != nil {
		panic(err)
	}

	err = mod.Start(ctx)
	defer mod.Close(ctx)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()
}

type static struct {
	gostream.VideoReader
}

func (s static) Close(_ context.Context) error {
	return nil
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}
}

func newCamera(ctx context.Context, _ resource.Dependencies, _ resource.Config, logger logging.Logger) (camera.Camera, error) {
	// ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=30 testsrc.mp4
	v, err := vidio.NewVideo("/usr/local/libs/camera-static/testsrc.mp4")
	handleErr(err)

	n := v.Frames()
	i := 0
	l := golog.NewLogger("newCamera")
	w, h := 640, 480
	encoder, _ := x264.NewEncoder(w, h, 30, l)

	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		defer func() { i = (i + 1) % n }()
		img, err := v.ReadFrames(i)
		handleErr(err)

		bytes, err := encoder.Encode(ctx, img[0])
		handleErr(err)
		return gimage.NewH264Image(bytes), func() {}, err
	})

	cam := static{
		VideoReader: reader,
	}

	src, err := camera.NewVideoSourceFromReader(ctx, cam, nil, camera.ColorStream)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get source from reader")
	}

	name := resource.NewName(camera.API, "static")
	return camera.FromVideoSource(name, src, logger), nil
}
