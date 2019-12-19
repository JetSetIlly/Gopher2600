// Package input represents the input/output part of the RIOT (the IO in RIOT).
// Note that the output aspect of the RIOT is minimal and so the package is
// called input.
//
// The main type in the package, the Input type, contains references to the
// three input devices - the panel and the two player ports. The player ports
// are only devices in the sense that they write to memory to specific
// addresses in a specific way. The actual devices that are connected via the
// player ports can of course, differ (joysticks, paddles, keyboards). Player
// ports are represented inside the emulation by the Player type.
//
// Connecting different devices to the Player type is handled in two ways. The
// simplest way is by calling the Handle() function of the  Player type.  The
// Handle() function takes a single Event argument and writes to memory
// accordingly. In this way, the host computer can translate input from
// physically connected devices into the correct Events. For example, the up
// cursor key on the keyboard might translate to an UP event to player zero.
//
// The second, more flexible way, is by Attach()ing an implementation of the
// Controller interface. The Controller interface requires one function,
// CheckInput(). The CheckInput() function is used to retrieve an Event, rather
// than having an Event pushed onto the Player. Internally, CheckInput() makes
// use of the Handle() function so it's just a question of what's more
// convenient.
//
// The input package also defines an EventRecorder interface. Every call to
// Handle(), either directly or via the Controller interface, causes
// RecordEvent() of an attached EventRecorder to be called. In effect, this
// causes a Handle() call to be mirrored somewhere else. In practice this
// interface is used by the Recorder type in the Recorder package. (The
// Playback type in the Recorder package meanwhile, uses the Controller
// interface to replay previously recorded events.)
//
// In addition to the Player port the input package also handles the VCS panel.
// The Panel type used for this works in exactly the same way as the Player
// type - a Handle() function that can be used directly and a Controller
// interface.
//
// This arrangement of Player & Panel types, Events, and Controller interfaces,
// gives great flexibility to the host computer. For instance, a single
// XBox style controller can be used to emulate both the digital joystick and
// the VCS panel. At the same time, the keyboard can be used to emulate the VCS
// panel and maybe the joystick for the second player.
package input
