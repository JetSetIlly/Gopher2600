package null

import (
	"gopher2600/gui"
)

type Null struct{}

func (n Null) SetMetaPixel(_ gui.MetaPixel) error {
	return nil
}

func (n Null) IsVisible() bool {
	return false
}

func (n Null) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	return nil
}

func (n Null) SetEventChannel(_ chan (gui.Event)) {
}
