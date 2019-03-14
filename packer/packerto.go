package packer

type PackerTo interface {
	PackTo(PackerCloser) error
}
