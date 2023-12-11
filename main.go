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
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/utils"
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

func (s *static) Name() resource.Name {
	return resource.NewName(camera.API, "static")
}

func (s *static) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

func (s *static) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (s *static) Projector(ctx context.Context) (transform.Projector, error) {
	//TODO implement me
	panic("implement me")
}

func (s *static) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	//TODO implement me
	panic("implement me")
}

func (s *static) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	//TODO implement me
	panic("implement me")
}

func (s *static) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	//TODO implement me
	panic("implement me")
}

func (s *static) Close(_ context.Context) error {
	return nil
}

func (s *static) Properties(_ context.Context) (camera.Properties, error) {
	return camera.Properties{
		MimeTypes: []string{
			utils.MimeTypeH264,
			utils.MimeTypeJPEG,
		},
	}, nil
}

func handleErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func frameToH264(ctx context.Context, e ourcodec.VideoEncoder, f image.Image) (image.Image, func(), error) {
	bytes, err := e.Encode(ctx, f)
	handleErr(err)

	return rimage.NewH264Image(bytes, 640, 480, 30), func() {}, err

}
func frameToJpeg(f *image.RGBA) (image.Image, func(), error) {
	b := new(bytes.Buffer)
	w := bufio.NewWriter(b)
	handleErr(rimage.EncodeJPEG(w, f))

	ret, err := jpeg.Decode(bytes.NewReader(b.Bytes()))
	handleErr(err)

	return ret, func() {}, err
}

func newCamera(_ context.Context, _ resource.Dependencies, _ resource.Config, _ logging.Logger) (camera.Camera, error) {
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
		if strings.Contains(mime, utils.MimeTypeH264) {
			fmt.Println("RETURNING H264")
			return frameToH264(ctx, encoder, frame)
		} else if strings.Contains(mime, utils.MimeTypeJPEG) {
			fmt.Println("RETURNING JPEG")
			return frameToJpeg(frame)
		} else {
			panic(fmt.Sprintln("unrecognized MIME type:", mime))
		}
	})

	cam := static{VideoReader: reader}
	handleErr(err)

	return &cam, nil
}
