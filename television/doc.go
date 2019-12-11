// Package television implements the output device of the emulated VCS. The
// television interface is used wherever a television needs to be connected.
// The NewTelevision() function creates a new instance of a reference
// implementation of the Television interface. In truth, it is probably the
// only implementation required but the option is there for alternatives.
//
// It is common for instances of television to be embedded in other type
// structure, thereby extending the "features" of the television and allowing
// the extended type to be used wherever the Television interface is required.
// The digest package is a good example of this idea.
//
// It is important to note that the reference television implementation does
// not render pixels or mix sound itself. Instead, the television interface
// exposes two functions, AddPixelRenderer() and AddAudioMixer(). These can be
// used to add as many renderers and mixers as required.
//
// The main means of communication is the Signal() function. This function
// accepts an instance of SignalAttributes which gives details of how the
// television should be behaving. The SignalAttributes type is meant to emulate
// the electrical signal between the VCS and the television but some
// concessions have been made for easement. The HSyncSimple in particular is a
// large piece of fudge but a necessary one.
package television
