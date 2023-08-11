package main

import (
	"camera-static/x264"
	"context"
	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	"github.com/viamrobotics/gostream"
	gimage "github.com/viamrobotics/gostream/image"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"image"
	"os"
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

func configFromImage(path string) image.Config {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		panic(err)
	}

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		panic(err)
	}
	return config
}

func imageFromPath(path string) image.Image {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	return img
}

const imgPath string = "/usr/local/libs/camera-static/sample_image.png"

func newCamera(ctx context.Context, _ resource.Dependencies, _ resource.Config, _ golog.Logger) (camera.Camera, error) {
	logger := golog.NewLogger("camera-static")

	config := configFromImage(imgPath)
	enc, err := x264.NewEncoder(config.Width, config.Height, 30, logger)
	if err != nil {
		panic(err)
	}

	img := imageFromPath(imgPath)
	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		bytes, err := enc.Encode(ctx, img)
		if err != nil {
			panic(err)
		}
		return gimage.NewH264Image(bytes), func() {}, nil
	})

	src, err := camera.NewVideoSourceFromReader(ctx, reader, nil, camera.ColorStream)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get source from reader")
	}

	name := resource.NewName(camera.API, "static")
	return camera.FromVideoSource(name, src), nil
}
