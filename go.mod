module github.com/janch32/fitkit-relay

go 1.13

require (
	github.com/albenik/go-serial/v2 v2.0.0
	github.com/janch32/fitkit-relay/discover v0.0.0
	github.com/janch32/fitkit-relay/fitkitbsl v0.0.0
	github.com/janch32/fitkit-relay/memory v0.0.0
	github.com/janch32/fitkit-relay/mspbsl v0.0.0
	github.com/marcinbor85/gohex v0.0.0-20180128172054-7a43cd876e46 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07 // indirect
)

replace github.com/janch32/fitkit-relay/discover => ./discover

replace github.com/janch32/fitkit-relay/fitkitbsl => ./fitkitbsl

replace github.com/janch32/fitkit-relay/mspbsl => ./mspbsl

replace github.com/janch32/fitkit-relay/memory => ./memory
