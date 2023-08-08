package main

import (
	"context"
	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	"github.com/viamrobotics/gostream"
	"go.viam.com/rdk/components/camera"
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
	logger := golog.NewLogger("static_camera")

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

func newCamera(ctx context.Context, _ resource.Dependencies, _ resource.Config, _ golog.Logger) (camera.Camera, error) {
	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		return image.NewYCbCr(image.Rect(0, 0, 640, 480), image.YCbCrSubsampleRatio420), func() {}, nil
	})

	cam := static{
		VideoReader: reader,
	}

	src, err := camera.NewVideoSourceFromReader(ctx, cam, nil, camera.ColorStream)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get source from reader")
	}

	name := resource.NewName(camera.API, "static")
	return camera.FromVideoSource(name, src), nil
}
