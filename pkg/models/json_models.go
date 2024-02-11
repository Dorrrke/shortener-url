// Пакет с описание моделей для запросов к базе данных и сериализации и десириализации в и из json.
package models

// RequestURLJson - модель для работы с запросом в теле которого приходит url для сокращения в формате json.
type RequestURLJson struct {
	URLAddres string `json:"url"`
}

// ResponseURLJson - модель для работы с ответом на запрос, в теле которого отправляется сокращенный url в формате json.
type ResponseURLJson struct {
	URLAddres string `json:"result"`
}

// RequestBatchURLModel - модель для работы с запросом в теле которого несколько url для сокращения в формате json.
type RequestBatchURLModel struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}

// ResponseBatchURLModel - модель для работы с ответом на запрос, в теле которого отправляется несколько сокращенных url в формате json.
type ResponseBatchURLModel struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"short_url"`
}

// URLModel - модель с полями в виде оригинального и сокращенного url для работы с бд и некоторыми хендлеами.
type URLModel struct {
	ShortID    string `json:"short_url"`
	OriginalID string `json:"original_url"`
}

type StatModel struct {
	URLsCount  int `json:"urls"`
	UsercCount int `json:"users"`
}

// BantchURL - для отправки на сохранение в базу данных нескольких скоращнных url сразу.
type BantchURL struct {
	OriginalURL string
	ShortURL    string
	UserID      string
}
