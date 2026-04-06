package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/vyacheslavskl/go-shorterner/internal/audit/mocks"
	models "github.com/vyacheslavskl/go-shorterner/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher_SubscribeAndNotify(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	observer1 := mocks.NewMockObserver(ctrl)
	observer2 := mocks.NewMockObserver(ctrl)

	publisher := NewPublisher()
	publisher.Subscribe(observer1)
	publisher.Subscribe(observer2)

	event := models.AuditEvent{
		Action: "shorten",
		UserID: "user-123",
	}

	// Ожидаем, что оба наблюдателя получат событие
	observer1.EXPECT().OnEvent(gomock.Any(), event).Return(nil)
	observer2.EXPECT().OnEvent(gomock.Any(), event).Return(nil)

	publisher.Notify(context.Background(), event)

	// Дадим горутинам время на выполнение
	time.Sleep(100 * time.Millisecond)
}

func TestPublisher_Notify_NoObservers(t *testing.T) {
	publisher := NewPublisher()
	event := models.AuditEvent{Action: "follow"}

	// Не должно быть паники или ошибок
	publisher.Notify(context.Background(), event)

	// Просто проверяем, что ничего не сломалось
	assert.True(t, true)
}

func TestFileObserver_NewFileObserver_EmptyPath(t *testing.T) {
	observer, err := NewFileObserver("")
	assert.Nil(t, observer)
	assert.NoError(t, err) // Возвращаем nil, но без ошибки (по логике кода)
}

func TestFileObserver_NewFileObserver_FileCreateError(t *testing.T) {
	// Используем путь, который почти наверняка вызовет ошибку (например, в корне /)
	observer, err := NewFileObserver("/invalid-path/forbidden-file.log")
	assert.Nil(t, observer)
	assert.Error(t, err)
}

func TestFileObserver_OnEvent_Success(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "audit_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	observer := &FileObserver{file: tmpfile, mu: sync.Mutex{}}

	event := models.AuditEvent{
		Action: AuditActionShorten,
		UserID: "user-456",
		URL:    "https://example.com",
	}

	err = observer.OnEvent(context.Background(), event)
	require.NoError(t, err)

	// Перечитываем файл
	data, err := os.ReadFile(tmpfile.Name())
	require.NoError(t, err)

	var loggedEvent models.AuditEvent
	err = json.Unmarshal(data[:len(data)-1], &loggedEvent) // убираем \n
	require.NoError(t, err)

	assert.Equal(t, event.Action, loggedEvent.Action)
	assert.Equal(t, event.UserID, loggedEvent.UserID)
	assert.Equal(t, event.URL, loggedEvent.URL)
}

func TestFileObserver_OnEvent_WriteError(t *testing.T) {
	// Мокаем *os.File, чтобы Write возвращал ошибку
	// badWriter := &BadWriter{}
	file := os.NewFile(1, "badfile") // некорректный дескриптор
	defer file.Close()

	observer := &FileObserver{file: file, mu: sync.Mutex{}}

	event := models.AuditEvent{Action: "shorten", UserID: "user"}

	err := observer.OnEvent(context.Background(), event)
	assert.Error(t, err)
}

// BadWriter — имитация сбоя при записи
type BadWriter struct{}

func (bw *BadWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

func TestHTTPObserver_OnEvent_RequestError(t *testing.T) {
	observer := &HTTPObserver{
		url:    ":invalid-url",
		client: &http.Client{},
	}

	event := models.AuditEvent{Action: "shorten"}

	err := observer.OnEvent(context.Background(), event)
	assert.Error(t, err)
}

func TestHTTPObserver_OnEvent_HTTPError(t *testing.T) {
	observer := &HTTPObserver{
		url: "http://localhost:1/xxx", // адрес, который почти наверняка не ответит
		client: &http.Client{
			Timeout: 1 * time.Millisecond,
		},
	}

	event := models.AuditEvent{Action: "follow"}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := observer.OnEvent(ctx, event)
	assert.Error(t, err)
	// Может быть timeout или connection refused
}

func TestPublisher_Notify_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	observer := mocks.NewMockObserver(ctrl)

	publisher := NewPublisher()
	publisher.Subscribe(observer)

	// Создаём контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем

	observer.EXPECT().OnEvent(ctx, gomock.Any()).Return(context.Canceled).AnyTimes()

	publisher.Notify(ctx, models.AuditEvent{Action: "test"})

	// Должно завершиться без паники
	time.Sleep(50 * time.Millisecond)
}
