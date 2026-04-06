package audit

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/vyacheslavskl/go-shorterner/internal/audit/mocks"
	models "github.com/vyacheslavskl/go-shorterner/internal/model"

	"github.com/stretchr/testify/assert"
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
