package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	vidio "github.com/AlexEidt/Vidio"
	"github.com/edaniels/golog"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	ourcodec "go.viam.com/rdk/gostream/codec"
	"go.viam.com/rdk/gostream/codec/x264"
	gimage "go.viam.com/rdk/gostream/image"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage"
	"image"
	"image/jpeg"
	"strings"
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
	handleErr(err)

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
		panic(err.Error())
	}
}

func frameToH264(ctx context.Context, e ourcodec.VideoEncoder, f image.Image) (image.Image, func(), error) {
	fmt.Println("H264 ENCODER!")
	bytes, err := e.Encode(ctx, f)
	handleErr(err)

	return gimage.NewH264Image(bytes), func() {}, err

}
func frameToJpeg(f *image.RGBA) (image.Image, func(), error) {
	fmt.Println("JPEG ENCODER!")
	b := new(bytes.Buffer)
	w := bufio.NewWriter(b)
	handleErr(rimage.EncodeJPEG(w, f))

	ret, err := jpeg.Decode(bytes.NewReader(b.Bytes()))
	handleErr(err)

	return ret, func() {}, err
}

func newCamera(ctx context.Context, _ resource.Dependencies, _ resource.Config, logger logging.Logger) (camera.Camera, error) {
	// ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=30 testsrc.mp4
	v, err := vidio.NewVideo("/usr/local/libs/camera-static/testsrc.mp4")
	handleErr(err)

	n, i := v.Frames(), 0
	w, h := 640, 480
	l := golog.NewLogger("newCamera")
	encoder, _ := x264.NewEncoder(w, h, 30, l)

	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		handleErr(v.ReadFrame(i))
		i = (i + 1) % n

		frame := image.NewRGBA(image.Rect(0, 0, w, h))
		frame.Pix = v.FrameBuffer()

		mime := gostream.MIMETypeHint(ctx, "")
		if strings.Contains(mime, "video/h264") {
			return frameToH264(ctx, encoder, frame)
		} else if strings.Contains(mime, "image/jpeg") {
			return frameToJpeg(frame)
		} else {
			panic(fmt.Sprintln("unrecognized MIME type:", mime))
		}
	})

	cam := static{VideoReader: reader}
	src, err := camera.NewVideoSourceFromReader(ctx, cam, nil, camera.ColorStream)
	handleErr(err)

	name := resource.NewName(camera.API, "static")
	return camera.FromVideoSource(name, src, logger), nil
}
