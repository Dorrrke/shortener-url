# Shortener

Сервис для сокращения ссылок

## Описание эндпоинтов

1. POST / - Сервер принимает в теле запроса строку URL как text/plain и возвращает ответ с кодом 201 и сокращённым URL как text/plain. При попытке пользователя сократить уже имеющийся в базе URL сервис возвращает HTTP-статус 409 Conflict, а в теле ответа — уже имеющийся сокращённый URL в правильном для хендлера формате. Если url удален возвращается статус 410 Gone
2. GET /{id} - где id — идентификатор сокращённого URL (например, /EwHXdJfB). В случае успешной обработки запроса сервер возвращает   ответ с кодом 307 и оригинальным URL в HTTP-заголовке Location.
3. POST /api/shorten - который принимает в теле запроса JSON-объект `{"url":"<some_url>"}` и возвращает в ответе объект `{"result":"<short_url>"}`. При попытке пользователя сократить уже имеющийся в базе URL сервис возвращает HTTP-статус 409 Conflict, а в теле ответа — уже имеющийся сокращённый URL в правильном для хендлера формате.
4. POST /api/shorten/batch - принимает в теле запроса множество URL для сокращения в формате:
``` 
[
    {
        "correlation_id": "<строковый идентификатор>",
        "original_url": "<URL для сокращения>"
    },
    ...
]
```
В качестве ответа хендлер возвращает данные в формате:
```
[
    {
        "correlation_id": "<строковый идентификатор из объекта запроса>",
        "short_url": "<результирующий сокращённый URL>"
    },
    ...
]
``` 
Все записи о коротких URL сохраняются в базе данных.
При попытке пользователя сократить уже имеющийся в базе URL сервис возвращает HTTP-статус 409 Conflict, а в теле ответа — уже имеющийся сокращённый URL в правильном для хендлера формате.
5. GET /api/user/urls - который возвращает пользователю все когда-либо сокращённые им URL в формате:
```
[
    {
        "short_url": "http://...",
        "original_url": "http://..."
    },
    ...
]
```
При отсутствии сокращённых пользователем URL хендлер отдавает HTTP-статус 204 No Content
6. DELETE /api/user/urls - который в теле запроса принимает список идентификаторов сокращённых URL для асинхронного удаления. Запрос может быть таким:
```
DELETE http://localhost:8080/api/user/urls
Content-Type: application/json

["6qxTVvsy", "RTfd56hn", "Jlfd67ds"]
```
В случае успешного приёма запроса хендлер возвращает HTTP-статус 202 Accepted. Фактический результат удаления происходит позже.
7. GET /ping - который при запросе проверяет соединение с базой данных. При успешной проверке хендлер возвращает HTTP-статус 200 OK, при неуспешной — 500 Internal Server Error

## Дополнительное описание функционала
Сервис выдает пользователю симметрично подписанную куку, содержащую уникальный идентификатор пользователя, если такой куки не существует или она не проходит проверку подлинности возвращается ошибка 401 Unauthorized.
Сервис конфигурируется с помощю ключей или переменных окружения:
* -a флаг конфигурирования адреса сервера
* -b флаг конфигурирования адреса запуска сокращенных url
* -f флаг для пути к файлу в который возможено сохранения url
* -d флаг содержащий данные базы данных для подключения
* SERVER_ADDRESS переменная окружения для конфигурирования адреса сервера
* BASE_URL переменная окружения конфигурирования адреса запуска сокращенных url
* FILE_STORAGE_PATH переменная окружения для пути к файлу в который возможено сохранения url
* DATABASE_DSN переменная окружения содержащий данные базы данных для подключения 

Хендлеры сервиса описаны тестами

## Библиотеки и тезнологии
Языки программирования: ![Go](https://img.shields.io/badge/-Go-0E2336?style=for-the-badge&logo=Go)
Библиотеки: Chi, pgx, Zap, env, JWT
