package stm

import (
	"context"
	"time"
)

type (
	Msg interface{}

	Cmd func() Msg

	State interface {
		Update(Msg) (State, Cmd)
		Init() Cmd
	}

	batched []Cmd

	// Stm is a state machine.
	Stm struct {
		messages chan Msg
		state    State

		ctx context.Context
	}

	StmOptions func(*Stm)
)

// default size of the message buffer.
const DefaultMessageBufferSize = 10

// Batch returns a command that will execute the given list of commands.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg {
		b := batched{}
		for _, cmd := range cmds {
			b = append(b, cmd)
		}
		return b
	}
}

// ToCmd returns a command that will send the given message immediatly.
func ToCmd(msg Msg) Cmd {
	return func() Msg {
		return msg
	}
}

// TransitionTo returns a `Cmd` and a `State` to transition to the given state,
// initializing it, calling the Init method of the given state and
// executing the given commands after the transition.
func TransitionTo(state State, cmds ...Cmd) (State, Cmd) {
	init := state.Init()
	cmds = append([]Cmd{init}, cmds...)
	return state, Batch(cmds...)
}

// Timer returns a command that will send the given message after the given
// duration.
func Timer(t time.Duration, timeExceedMessage Msg) Cmd {
	return func() Msg {
		if t < 0 {
			t = 0
		}
		timer := time.NewTimer(t)
		<-timer.C
		return timeExceedMessage
	}
}

func (stm *Stm) loop() {
	for {
		select {

		case <-stm.ctx.Done():
			return

		case msg := <-stm.messages:
			var cmd Cmd
			stm.state, cmd = stm.state.Update(msg)
			if cmd != nil {
				stm.Send(cmd)
			}
		}
	}
}

// Send a command to the state machine. Note that the execution of the command
// is done in a goroutine and therefore the order of execution is not guaranteed.
func (stm *Stm) Send(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if msg == nil {
			return
		}

		if batch, ok := msg.(batched); ok {
			// recursively send all commands in the batch
			for _, batchCmd := range batch {
				stm.Send(batchCmd)
			}

		} else {
			stm.messages <- msg
		}
	}()
}

// New creates and starts a state machine with the initial state and options.
// The state machine will be terminated when the context is done.
func New(ctx context.Context, initialState State, opts ...StmOptions) *Stm {
	stm := &Stm{
		messages: make(chan Msg, DefaultMessageBufferSize),
		state:    initialState,
		ctx:      ctx,
	}

	for _, opt := range opts {
		opt(stm)
	}

	go stm.loop()
	return stm
}

// WithMessageBufferSize sets the size of the message buffer
func WithMessageBufferSize(size int) StmOptions {
	return func(stm *Stm) {
		stm.messages = make(chan Msg, size)
	}
}
