package storage

type URLCreatorGetter interface {
	CreateURL(URLId string, origURL string)
	GetOrigURL(URLId string) string
	CheckMapKey(URLId string) bool
}

type URLStorage struct {
	URLMap map[string]string
}

var MapURL URLStorage

func (storage URLStorage) CreateURL(URLId string, origURL string) {
	storage.URLMap[URLId] = origURL
}

func (storage URLStorage) GetOrigURL(URLId string) string {
	return storage.URLMap[URLId]
}

func (storage URLStorage) CheckMapKey(URLId string) bool {
	if _, ok := storage.URLMap[URLId]; ok {
		return true
	} else {
		return false
	}
}
