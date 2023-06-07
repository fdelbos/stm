package stm_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	. "github.com/fdelbos/stm"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	ctx context.Context
}

func (s *Suite) randString() string {
	buff := make([]byte, 8)
	_, err := rand.Reader.Read(buff)
	s.Require().NoError(err)
	return hex.EncodeToString(buff)
}

func TestSuite(t *testing.T) {
	suite.Run(t, &Suite{})
}

func (s *Suite) SetupTest() {
	s.ctx = context.Background()
}

func (s *Suite) TestStm() {
	state := mocks.NewStmState(s.T())
	var machine *Stm
	ctx, cancel := context.WithCancel(s.ctx)

	s.Run("should initialize the state machine", func() {
		machine = New(ctx, state)
		s.NotNil(machine)
	})

	s.Run("should send a message to the state machine", func() {
		chNotif := make(chan Msg, 1)
		msg := s.randString()

		state.On("Update", msg).Return(func(msg Msg) (State, Cmd) {
			chNotif <- msg
			return state, nil
		})
		machine.Send(ToCmd(msg))
		<-chNotif
	})

	s.Run("should send a messages back", func() {
		chNotif := make(chan Msg, 1)
		defer close(chNotif)
		msg1 := s.randString()
		msg2 := s.randString()

		state.On("Update", msg1).Return(func(msg Msg) (State, Cmd) {
			return state, ToCmd(msg2)
		})

		state.On("Update", msg2).Return(func(msg Msg) (State, Cmd) {
			chNotif <- msg2
			return state, nil
		})

		machine.Send(ToCmd(msg1))
		s.Equal(msg2, <-chNotif)
	})

	s.Run("should send a batch of messages", func() {
		chNotif := make(chan interface{})
		msg1 := s.randString()
		msg2 := s.randString()

		state.On("Update", msg1).Return(func(received Msg) (State, Cmd) {
			chNotif <- nil
			return state, nil
		})

		state.On("Update", msg2).Return(func(received Msg) (State, Cmd) {
			chNotif <- nil
			return state, nil
		})

		machine.Send(Batch(ToCmd(msg2), ToCmd(msg1)))

		<-chNotif
		<-chNotif
	})

	s.Run("should send a message after some time", func() {
		duration := time.Millisecond * 200
		chNotif := make(chan Msg, 1)
		msg := s.randString()

		state.On("Update", msg).Return(func(msg Msg) (State, Cmd) {
			chNotif <- msg
			return state, nil
		})

		start := time.Now()
		machine.Send(Timer(duration, msg))
		s.Equal(<-chNotif, msg)
		s.True(time.Since(start) > duration)
	})

	s.Run("should transition to a new state", func() {
		chNotif := make(chan interface{})
		msgInit := s.randString()
		msgAfter := s.randString()

		newState := mocks.NewStmState(s.T())
		newState.On("Init").Return(func() Cmd {
			return ToCmd(msgInit)
		})

		newState.On("Update", msgInit).Return(func(Msg) (State, Cmd) {
			chNotif <- nil
			return newState, nil
		})

		newState.On("Update", msgAfter).Return(func(Msg) (State, Cmd) {
			chNotif <- nil
			return newState, nil
		})

		state.On("Update", "start").Return(func(Msg) (State, Cmd) {
			state, cmd := TransitionTo(newState, ToCmd(msgAfter))
			return state, cmd
		})

		machine.Send(ToCmd("start"))

		// message can be sent in any order so lets wait a bit
		timer := time.NewTimer(time.Millisecond * 100)
		<-timer.C
		<-chNotif
		<-chNotif
	})

	s.Run("should send a nil message", func() {
		machine.Send(nil)
		machine.Send(ToCmd(nil))
		timer := time.NewTimer(time.Millisecond * 200)
		<-timer.C
	})

	s.Run("should terminate the state machine", func() {
		cancel()
	})

}
