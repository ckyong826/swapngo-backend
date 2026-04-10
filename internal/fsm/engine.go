package fsm

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrIdempotentState   = errors.New("target state already reached (idempotent)")
)

// Transition 定义了状态流转规则
type Transition struct {
	From  string
	Event string
	To    string
}

// StateMachine 是一个通用的状态机引擎
type StateMachine struct {
	mu          sync.RWMutex
	transitions map[string]map[string]string // map[FromState]map[Event]ToState
}

// New 实例化一个状态机，并注册所有合法的流转规则
func New(rules []Transition) *StateMachine {
	sm := &StateMachine{
		transitions: make(map[string]map[string]string),
	}
	for _, rule := range rules {
		if _, ok := sm.transitions[rule.From]; !ok {
			sm.transitions[rule.From] = make(map[string]string)
		}
		sm.transitions[rule.From][rule.Event] = rule.To
	}
	return sm
}

// Fire 触发事件，返回目标状态。
// 它不负责修改数据库，只负责纯粹的逻辑校验。
func (sm *StateMachine) Fire(currentState, event string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowedEvents, ok := sm.transitions[currentState]
	if !ok {
		return "", fmt.Errorf("%w: state %s has no outgoing transitions", ErrInvalidTransition, currentState)
	}

	nextState, ok := allowedEvents[event]
	if !ok {
		return "", fmt.Errorf("%w: cannot trigger event %s from state %s", ErrInvalidTransition, event, currentState)
	}

	return nextState, nil
}