# Org API - API организационной структуры

REST API для управления подразделениями и сотрудниками. Построен на Go (net/http), GORM, PostgreSQL. Миграции через goose.


## Быстрый старт

### Требования

- Docker >= 24
- Docker Compose >= 2

### Запуск

```bash
git https://github.com/ArslanIsaev/Organizational-Structure-API
cd org-api
docker-compose up --build
```

API будет доступно по адресу: http://localhost:8080

При запуске приложение автоматически:
1. Дожидается готовности PostgreSQL (healthcheck)
2. Применяет все миграции через goose
3. Запускает HTTP-сервер


## Запуск тестов

Тесты используют SQLite in-memory - PostgreSQL **не нужен**.

```bash
go test ./tests/... -v -count=1
```


## Архитектура

```
org-api/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа
├── internal/
│   ├── config/                  # Конфигурация из переменных окружения
│   ├── db/                      # Подключение к PostgreSQL через GORM
│   ├── model/                   # Модели данных 
│   ├── repository/              # Слой работы с БД 
│   ├── service/                 # Бизнес-логика, валидация, обработка ошибок
│   ├── handler/                 # HTTP-хендлеры, парсинг запросов и ответов
│   └── middleware/              # Логгирование запросов
├── migrations/                  # SQL-миграции goose
├── tests/                       # Тесты 
├── Dockerfile                   
├── docker-compose.yml
└── README.md
```

**Слои и их ответственность:**

| Слой | Пакет | Отвечает за |
|------|-------|-------------|
| Model | internal/model | Структуры данных, GORM-теги |
| Repository | internal/repository | Запросы к БД, CTE для дерева |
| Service | internal/service | Бизнес-правила, валидация, обработка ошибок |
| Handler | internal/handler | HTTP: парсинг, роутинг, форматирование ответа |



## API Reference

### 1. Создать подразделение

```
POST /departments/
```

**Тело запроса:**

```json
{
  "name": "Engineering",
  "parent_id": null
}
```

**Ответ 201 Created:**

```json
{
  "id": 1,
  "name": "Engineering",
  "parent_id": null,
  "created_at": "2026-05-23T17:00:00Z"
}
```

**Ошибки:**
- 400 - пустое имя или имя длиннее 200 символов
- 404 - указанный parent_id не существует
- 409 - подразделение с таким именем уже есть у этого родителя



### 2. Создать сотрудника в подразделении

```
POST /departments/{id}/employees/
```

**Тело запроса:**

```json
{
  "full_name": "Иван Иванов",
  "position": "Team Lead",
  "hired_at": "2023-06-01"
}
```


**Ответ 201 Created:**

```json
{
  "id": 1,
  "department_id": 1,
  "full_name": "Иван Иванов",
  "position": "Team Lead",
  "hired_at": "2023-06-01T00:00:00Z",
  "created_at": "2026-05-23T17:00:00Z"
}
```

**Ошибки:**
- 400 - пустые full_name или position
- 404 - подразделение не найдено



### 3. Получить подразделение дерево + сотрудники

```
GET /departments/{id}?depth=1&include_employees=true&sort_by=full_name
```

**Query-параметры:**

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|-------------|----------|
| depth | int | 1 | Глубина вложенных подразделений (макс 5) |
| include_employees | bool | true | Включать ли сотрудников в ответ |
| sort_by | string | created_at | Сортировка сотрудников: created_at или full_name |

**Ответ 200 OK:**

```json
{
  "id": 1,
  "name": "Engineering",
  "parent_id": null,
  "created_at": "2026-05-23T17:00:00Z",
  "employees": [
    {
      "id": 1,
      "department_id": 1,
      "full_name": "Иван Иванов",
      "position": "Team Lead",
      "hired_at": "2023-06-01T00:00:00Z",
      "created_at": "2026-05-23T17:00:00Z"
    }
  ],
  "children": [
    {
      "id": 2,
      "name": "Backend",
      "parent_id": 1,
      "created_at": "2026-05-23T17:01:00Z",
      "employees": [],
      "children": []
    }
  ]
}
```

**Ошибки:**
- 400 - некорректный depth
- 404 - подразделение не найдено


### 4. Переместить / переименовать подразделение

```
PATCH /departments/{id}
```

**Тело запроса** (все поля опциональны):

```json
{
  "name": "Platform Engineering",
  "parent_id": 3
}
```

Для отвязки от родителя передайте `"parent_id": null`

**Ответ 200 OK:** обновлённое подразделение

**Ошибки:**
- 400 - некорректные данные
- 404 - подразделение или новый родитель не найден
- 409 - попытка сделать подразделение родителем самого себя или создать цикл в дереве



### 5. Удалить подразделение

```
DELETE /departments/{id}?mode=cascade
DELETE /departments/{id}?mode=reassign&reassign_to_department_id=5
```

**Query-параметры:**

| Параметр | Обязателен | Описание |
|----------|-----------|----------|
| mode | да | cascade или reassign |
| reassign_to_department_id | при mode=reassign | ID подразделения для перевода сотрудников |

**Режимы удаления:**

- **cascade** - удаляет подразделение, всех его сотрудников и все дочерние подразделения рекурсивно
- **reassign** - переводит сотрудников удаляемого подразделения и всех дочерних в указанное, затем удаляет их

**Ответ 200 OK:**

```json
{ "status": "deleted" }
```

**Ошибки:**
- 400 - не указан mode или отсутствует reassign_to_department_id при mode=reassign
- 404 - подразделение не найдено или не найден reassign_to_department_id



## Примеры curl

```bash
# Создать корневое подразделение
curl -s -X POST http://localhost:8080/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name": "Engineering"}' | jq

# Создать дочернее
curl -s -X POST http://localhost:8080/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name": "Backend", "parent_id": 1}' | jq

# Добавить сотрудника
curl -s -X POST http://localhost:8080/departments/1/employees/ \
  -H "Content-Type: application/json" \
  -d '{"full_name": "Иван Иванов", "position": "Team Lead", "hired_at": "2023-06-01"}' | jq

# Получить дерево глубиной 3
curl -s "http://localhost:8080/departments/1?depth=3&include_employees=true" | jq

# Переименовать подразделение
curl -s -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"name": "Platform"}' | jq

# Переместить в другое подразделение
curl -s -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"parent_id": 3}' | jq

# Удалить каскадно
curl -s -X DELETE "http://localhost:8080/departments/2?mode=cascade" | jq

# Удалить с переводом сотрудников
curl -s -X DELETE "http://localhost:8080/departments/2?mode=reassign&reassign_to_department_id=1" | jq
```


## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| DB_HOST | localhost | Хост PostgreSQL |
| DB_PORT | 5432 | Порт PostgreSQL |
| DB_USER | orgapi | Пользователь БД |
| DB_PASSWORD | orgapi_secret | Пароль БД |
| DB_NAME | orgapi_db | Имя базы данных |
| SERVER_PORT | 8080 | Порт HTTP-сервера |



## Технологии

| Технология | Версия | Применение |
|-----------|--------|-----------|
| Go | 1.22 | Язык реализации |
| net/http | stdlib | HTTP-сервер без фреймворков |
| GORM | 1.25 | ORM для работы с БД |
| goose | 3.21 | SQL-миграции |
| PostgreSQL | 16 | База данных |
| Docker / Compose | - | Контейнеризация |
| SQLite (тесты) | - | In-memory БД для тестов |
