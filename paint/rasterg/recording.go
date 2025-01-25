package main

import (
	"sync/atomic"
)

type ShaderId int

type ResourceId uint64

func (r *ResourceId) next() {
	atomic.AddUint64((*uint64)(r), 1)
}

type Recording struct {
	commands []Command
}

type BufferProxy struct {
	size uint64
	id   ResourceId
	name string
}

type ImageFormat int

const (
	Rgba8 ImageFormat = iota
	Bgra8
)

type ImageProxy struct {
	width  uint32
	height uint32
	format ImageFormat
	id     ResourceId
}

type ResourceProxy interface{}

type Command struct {
}

type BindType int

const (
	Buffer BindType = iota
	BufReadOnly
	Uniform
	Image
	ImageRead
)

type DrawParams struct {
	shaderId      ShaderId
	instanceCount uint32
	vertexCount   uint32
	vertexBuffer  *BufferProxy
	resources     []ResourceProxy
	target        *ImageProxy
	clearColor    *[4]float32
}

func (r *Recording) push(cmd Command) {
	r.commands = append(r.commands, cmd)
}

func (r *Recording) upload(name string, data []byte) *BufferProxy {
	bufProxy := &BufferProxy{
		size: uint64(len(data)),
		name: name,
	}
	r.push(Command{})
	return bufProxy
}

func (r *Recording) uploadUniform(name string, data []byte) *BufferProxy {
	bufProxy := &BufferProxy{
		size: uint64(len(data)),
		name: name,
	}
	r.push(Command{})
	return bufProxy
}

func (r *Recording) uploadImage(width, height uint32, format ImageFormat, data []byte) *ImageProxy {
	imageProxy := &ImageProxy{
		width:  width,
		height: height,
		format: format,
	}
	r.push(Command{})
	return imageProxy
}

func (r *Recording) writeImage(proxy *ImageProxy, x, y uint32, image Image) {
	r.push(Command{})
}

func (r *Recording) dispatch(shader ShaderId, wgSize [3]uint32, resources []ResourceProxy) {
	r.push(Command{})
}

func (r *Recording) dispatchIndirect(shader ShaderId, buf *BufferProxy, offset uint64, resources []ResourceProxy) {
	r.push(Command{})
}

func (r *Recording) draw(params DrawParams) {
	r.push(Command{})
}

func (r *Recording) download(buf *BufferProxy) {
	r.push(Command{})
}

func (r *Recording) clearAll(buf *BufferProxy) {
	r.push(Command{})
}

func (r *Recording) freeBuffer(buf *BufferProxy) {
	r.push(Command{})
}

func (r *Recording) freeImage(image *ImageProxy) {
	r.push(Command{})
}

func (r *Recording) freeResource(resource ResourceProxy) {
	r.push(Command{})
}

func (r *Recording) intoCommands() []Command {
	return r.commands
}

func NewBufferProxy(size uint64, name string) *BufferProxy {
	id := ResourceId(1)
	return &BufferProxy{
		size: size,
		id:   id,
		name: name,
	}
}

func NewImageProxy(width, height uint32, format ImageFormat) *ImageProxy {
	id := ResourceId(1)
	return &ImageProxy{
		width:  width,
		height: height,
		format: format,
		id:     id,
	}
}
