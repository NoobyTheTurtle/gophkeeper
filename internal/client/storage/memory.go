package storage

import (
	"errors"
	"sync"

	"github.com/smanhack/gophkeeper/internal/client/model"
	proto "github.com/smanhack/gophkeeper/pkg/api"
)

type MemoryStorage struct {
	mu               sync.RWMutex
	LoginPassSecrets map[int]model.LoginPassSecret
	TextSecrets      map[int]model.TextSecret
	CardSecrets      map[int]model.CardSecret
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		LoginPassSecrets: make(map[int]model.LoginPassSecret, 0),
		TextSecrets:      make(map[int]model.TextSecret, 0),
		CardSecrets:      make(map[int]model.CardSecret, 0),
	}
}

func (ms *MemoryStorage) GetLoginPassSecret(id int) (model.LoginPassSecret, bool, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, ok := ms.LoginPassSecrets[id]
	if !ok {
		return model.LoginPassSecret{}, ok, errors.New("Login/Pass not found")
	}

	return data, ok, nil
}

func (ms *MemoryStorage) SetLoginPassSecrets(models []model.LoginPassSecret) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, m := range models {
		ms.LoginPassSecrets[m.Id] = m
	}
}

func (ms *MemoryStorage) ResetStorage() {
	ms.LoginPassSecrets = make(map[int]model.LoginPassSecret, 0)
	ms.TextSecrets = make(map[int]model.TextSecret, 0)
	ms.CardSecrets = make(map[int]model.CardSecret, 0)
}

func (ms *MemoryStorage) GetCardSecret(id int) (model.CardSecret, bool, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, ok := ms.CardSecrets[id]
	if !ok {
		return model.CardSecret{}, ok, errors.New("card data not found")
	}

	return data, ok, nil
}

func (ms *MemoryStorage) SetTextSecrets(models []model.TextSecret) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, m := range models {
		ms.TextSecrets[m.Id] = m
	}
}

func (ms *MemoryStorage) GetTextSecret(id int) (model.TextSecret, bool, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, ok := ms.TextSecrets[id]
	if !ok {
		return model.TextSecret{}, ok, errors.New("text not found")
	}

	return data, ok, nil
}

func (ms *MemoryStorage) SetCardSecrets(models []model.CardSecret) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, m := range models {
		ms.CardSecrets[m.Id] = m
	}
}

func (ms *MemoryStorage) FindInStorage(id int) (interface{}, bool) {
	data, ok, _ := ms.GetLoginPassSecret(id)
	if ok {
		return data, true
	}

	dataCard, okCard, _ := ms.GetCardSecret(id)
	if okCard {
		return dataCard, true
	}

	textData, okText, _ := ms.GetTextSecret(id)
	if okText {
		return textData, true
	}

	return nil, false
}

func (ms *MemoryStorage) GetDataRecordList(id int) []*proto.DataRecord {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var list []*proto.DataRecord
	switch id {
	case 1:
		for _, data := range ms.LoginPassSecrets {
			list = append(list, &proto.DataRecord{Id: uint32(data.Id), Name: data.Name})
		}
	case 2:
		for _, data := range ms.TextSecrets {
			list = append(list, &proto.DataRecord{Id: uint32(data.Id), Name: data.Name})
		}
	case 4:
		for _, data := range ms.CardSecrets {
			list = append(list, &proto.DataRecord{Id: uint32(data.Id), Name: data.Name})
		}
	}

	return list
}
