package bot

import "sync"

type UserState int

const (
	StateNone UserState = iota
	StateWaitFullName
	StateWaitDocs
	StateAdminWaitRejectReason
)

type Session struct {
	UserID       int64
	GroupID      int64
	FullName     string
	MediaMsgIDs  []int
	ConfirmMsgID int

	RejectTargetUserID int64
	RejectTargetChatID int64
}

type StateManager struct {
	mu      sync.RWMutex
	states  map[int64]UserState
	session map[int64]*Session
}

func NewStateManager() *StateManager {
	return &StateManager{
		states:  make(map[int64]UserState),
		session: make(map[int64]*Session),
	}
}

func (sm *StateManager) SetState(userID int64, state UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[userID] = state
}

func (sm *StateManager) GetState(userID int64) UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.states[userID]
}

func (sm *StateManager) SetSession(userID int64, s *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.session[userID] = s
}

func (sm *StateManager) GetSession(userID int64) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.session[userID]
	return s, ok
}

func (sm *StateManager) Clear(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.states, userID)
	delete(sm.session, userID)
}
