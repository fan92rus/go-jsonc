# Архитектурное ревью, Code Review, Production Readiness

> Дата: 2025-07-24
> Проект: github.com/fan92rus/go-jsonc
> Ветка: master (b2ee4b3)
> Язык: Go 1.25

---

## 1. Архитектурное ревью

### 1.1 Структура пакета

```
jsonc-cst/
├── node.go         # 328 строк — CST Node, NodeKind, Position, методы навигации
├── lexer.go        # 269 строк — токены + лексер
├── parser.go       # 257 строк — парсер (struct + Parse + parse*)
├── serialize.go    #  30 строк — Serialize
├── format.go       # 252 строк — Format, FormatOptions, fmtNode/fmtContainer/fmtMember
├── gen_test.go     # 313 строк — PBT-генераторы (rapid)
├── property_test.go # 1156 строк — PBT-тесты
├── example_test.go #  65 строк — примеры для godoc
├── .golangci.yml   # 56 строк — 22 линтера
├── .github/        # CI (2 Go версии × 2 OS + golangci-lint)
├── scripts/        # pre-commit hook
└── go.mod / go.sum
```

**Оценка: 🟢 Отлично**

Разделение на 4 исходных файла с чёткими зонами ответственности:
- **lexer.go** — токенизация. Никаких знаний о CST-дереве.
- **parser.go** — токены → CST. Ничего не знает о форматировании/сериализации.
- **serialize.go** — CST → сырой текст. Минимальный, 30 строк.
- **format.go** — CST → pretty-printed текст. Единственный файл с нетривиальной логикой (5 helper-функций).
- **node.go** — типы данных и методы навигации.

Зависимости: `lexer.go → parser.go → serialize.go / format.go`. Все в одном пакете — циклических зависимостей нет.

### 1.2 Data Flow

```
Input []byte
    ↓
lexer.next() → token{kind, text, pos}
    ↓
parser.parse*() → Node{CST с Kids + Value + Position}
    ↓
Serialize → string (потеря форматирования, сохранение структуры)
Format   → string (канонический pretty-print)
Walk     → visitor pattern
FindAll  → фильтрация по Kind
```

### 1.3 Структура данных: CST vs AST

CST — правильный выбор для пакета, работающего с JSONC. Отличия:
- **AST** теряет комментарии, пробелы, порядок скобок.
- **CST** сохраняет всё — комментарии, каждый байт исходника.

`KindLBrace` / `KindRBrace` / `KindColon` / `KindComma` / `KindWhitespace` — hallmark
настоящего CST. Каждый токен исходника представлен в дереве.

**Вердикт:** Архитектура корректная и хорошо продуманная. Для JSONC — именно CST, не AST.

### 1.4 Явные архитектурные решения (оценка)

| Решение | Оценка | Комментарий |
|---------|--------|-------------|
| Error Node recovery | ✅ | Ошибки — узлы в дереве, а не panic/return err. Критично для редакторов/IDE. |
| `Parse()` never returns err | ⚠️ | `err` всегда nil (кроме nil-doc). Путает пользователя. |
| Token-буфер в парсере | ✅ | `peek() + advance()` — классика. |
| `Position` с Offset/Line/Column | ✅ | Все три поля — необходимый минимум. |
| Комментарии как `KindComment` | ✅ | `CommentStyle` + `CommentBody` отдельно от `Value`. |
| Фильтр `filterNonTriviaCST` | ✅ | Правильно отличает комментарии/пробелы от значимых узлов. |

---

## 2. Code Review

### 2.1 Стиль и качество кода

- `golangci-lint` — **0 issues** с 22 линтерами, включая `gocyclo` (20), `funlen` (60/40), `gocognit` (30), `dupl` (100).
- `go vet` — чист.
- `go test -race` — проходит на обеих ОС (в CI).
- Godoc — на всех экспортируемых типах и функциях.

**Статистика:** 2670 строк Go, 1156 строк тестов = 43% кода — тесты. Отличное соотношение.

### 2.2 Конкретные замечания

#### issue 1: `Parse()` — вводящая в заблуждение сигнатура

```go
func Parse(input []byte) (*Node, error)
```

Возвращает `err != nil` **только** когда `doc == nil`, что на практике не случается —
вместо ошибки всегда генерируется `KindError`-node в дереве. Пользователь пишет:

```go
doc, err := jsonc.Parse(data)
if err != nil { log.Fatal(err) } // этот код НИКОГДА не выполняется
```

**Рекомендация:** Либо уберите `error` из возврата: `func Parse(input []byte) *Node`,
либо сделайте `error` осмысленным (критические ошибки вроде memory).

#### issue 2: `scanNumber` принимает невалидные последовательности

```go
func (l *lexer) scanNumber(start Position) token {
    for l.pos < len(l.input) {
        c := l.input[l.pos]
        if c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' || (c >= '0' && c <= '9') {
            l.advance()
        } else { break }
    }
```

Этот код примет: `--1`, `+-2`, `1e`, `1.2.3`, `1-2`, `1+2`, `1e--5`.

JSON spec (RFC 8259) допускает ровно один опциональный `-` в начале, один `.`,
один `e`/`E` с опциональным `+`/`-` в экспоненте. Текущий код пропускает всё.

**Severity:** Medium — PBT тесты покрывают валидные числа, но сканер не отклоняет невалидные.

**Рекомендация:** Добавить validate-фазу после сканирования (или переписать сканер
по автоматному принципу с конечным числом состояний). Текущий PBT
`TestProperty_RejectInvalidJSON` не тестирует эти случаи.

#### issue 3: `scanKeyword` не восстанавливает позицию при неудаче

```go
func (l *lexer) scanKeyword(expected string, kind tokenKind, start Position) token {
    end := start.Offset + len(expected)
    if end > len(l.input) || string(l.input[start.Offset:end]) != expected {
        return token{kind: tokError, ...}
    }
```

При неудаче `l.pos` не сброшен — позиция съедена. Это ок, потому что контент уже
прочитан (первый символ считан в `scanValueToken`), но поведение недокументировано.
Для `truee` лексер выдаст `tokError` + последующие токены (в нашем случае `e` остаётся
в сканере и будет обработан в следующем `next()`).

Проверка: на входе `truee` выдаётся `tokError`, и дополнительный `e` — как `tokEOF`.
Не баг, но хрупко.

#### issue 4: `fmtContainer` — дублирование в `hasComments` / `hasMembers`

В `fmtContainer` два прохода по детям: сначала `children := filterNonTriviaCST(n.Children)`,
потом отдельный цикл `for _, c := range n.Children { if c.Kind == KindComment { ... } }`.

Можно объединить в один проход.

**Severity:** Low — косметика.

#### issue 5: Pre-commit hook — race condition

```sh
golangci-lint run ./... > /tmp/golangci-lint-out.txt 2>&1
```

Использует `/tmp/...` без уникального суффикса. Два параллельных commit'а
(например, через worktree'ы) будут перезаписывать вывод друг друга.

**Severity:** Low — только локально, только при параллельных коммитах.

#### issue 6: `testdata/` в `.gitignore`

`testdata/` перечислена в `.gitignore`, что нестандартно — Go tools ожидают
`testdata/` как место для тестовых данных. Сейчас это не проблема (генераторы
в коде), но может сбить с толку.

#### issue 7: `go.sum` — `rapid` помечен как `// indirect`

В `go.mod` написано `// indirect` для `rapid`, хотя он используется напрямую
в `*_test.go` файлах. Это может вызвать вопросы при `go mod tidy`.

### 2.3 PBT-покрытие

| Категория | Число тестов | Тип |
|-----------|-------------|------|
| Parse valid JSON | 4 | PBT |
| Parse valid JSONC | 3 | PBT |
| Container structure | 3 | PBT |
| Comment preservation | 5 | PBT |
| Position tracking | 2 | PBT |
| Serialization | 2 | PBT |
| Formatting | 5 | PBT + table |
| Error handling | 3 | PBT + table |
| Edge cases | 6 | PBT + table |

**Всего: ~31 PBT + 8 table-driven** = 45 PASS-строк.

**Пробелы в PBT-покрытии:**
- Нет тестов для невалидных чисел (`--1`, `1.2.3`, `1e`)
- Нет тестов на пустой/whitespace-only input
- Нет тестов на `Format(doc, nil)` vs `Format(doc, &FormatOptions{})`
- Нет тестов на `Node.String()` / debug representation
- Нет тестов на `DeepEqual` с комментариями
- Нет тестов на `Elements()` / `Members()` с пустыми контейнерами

---

## 3. Production Readiness

### 3.1 Чеклист production-ready

| Критерий | Статус | Комментарий |
|----------|--------|-------------|
| **Тесты проходят** | ✅ `go test -race ./...` — зелёный |
| **Линтер** | ✅ 0 issues, 22 линтера |
| **CI** | ✅ 2 Go версии, 2 OS, golangci-lint |
| **Pre-commit** | ✅ Автоматический прогон |
| **Error handling** | ✅ Error-recovery nodes |
| **Panic-free** | ✅ 0 panics даже на случайном вводе |
| **Deep nesting** | ✅ Проверен до 500+ уровней |
| **Unicode** | ✅ Базовое покрытие |
| **Документация** | ✅ README, godoc, примеры |
| **Лицензия** | ✅ MIT |
| **Version tag** | ❌ **Нет git tag** — `go get` без версии |
| **Go модуль** | ✅ `go.mod` с module path |
| **Go minimum version** | ⚠️ 1.25 — очень новая (июль 2025). В CI только 1.24+. |

### 3.2 Критические блокеры для production

**1. Version tag отсутствует.** Без `v1.0.0` (или хотя бы `v0.1.0`)
пользователь не может зафиксировать версию через `go get`:

```bash
go get github.com/fan92rus/go-jsonc@latest
# Получит HEAD master — может сломаться в любой момент
```

Нужно создать и опубликовать тег:
```bash
git tag v0.1.0 && git push origin v0.1.0
```

**2. Go 1.25 — риски.** `go 1.25.0` в `go.mod`. Это bleeding-edge.
Многие пользователи сидят на 1.22/1.23. Проверка CI на 1.24 есть, но
`go.mod` заявляет 1.25. Стоит понизить до 1.22+.

**3. `Parse()` без ошибок.** Пользователь привык проверять `err` на nil.
Пакет, где `err` никогда не заполняется, создаёт false sense of security.
Либо убрать `error` из возврата, либо сделать осмысленным.

### 3.3 Performance

- Сборка: 1.7s (полный тест-сьют). 
- Глубокие деревья (500 уровней) — без stack overflow (рекурсивный парсер без tail-call).
- Быстродействие: не бенчмаркнуто. Нет `*_test.go` бенчмарков.
- Allocation profile: неизвестен.

Для production config-файлов (100-500 строк) производительность более чем достаточна.

---

## 4. API Readiness Review

### 4.1 Публичный API

```go
// Parsing
func Parse(input []byte) (*Node, error)     // ⚠️ error всегда nil

// Serialization
func Serialize(doc *Node) string

// Formatting
type FormatOptions struct { Indent string }
func Format(doc *Node, opts *FormatOptions) string

// Types
type NodeKind int         // + 18 Kind* констант
type Position struct { ... }  // Offset, Line, Column
type CommentStyle int     // + CommentLine, CommentBlock
type Node struct { ... }  // Kind, Children, Value, CommentStyle, CommentBody, Start, End

// Node methods
func (n *Node) RawText() string
func (n *Node) IsContainer() bool
func (n *Node) IsValue() bool
func (n *Node) IsTrivia() bool
func (n *Node) Walk(fn func(*Node) bool)
func (n *Node) FindAll(kind NodeKind) []*Node
func (n *Node) FindAllComments() []*Node
func (n *Node) FirstChild() *Node
func (n *Node) FirstChildOfKind(kinds ...NodeKind) *Node
func (n *Node) ValueNode() *Node
func (n *Node) KeyNode() *Node
func (n *Node) Members() []*Node
func (n *Node) Elements() []*Node
func (n *Node) DeepEqual(other *Node) bool
func (n *Node) String() string
func (k NodeKind) String() string
func (p Position) String() string
```

### 4.2 Что есть

- ✅ Parse → CST
- ✅ CST → Serialize (lossless)
- ✅ CST → Format (pretty-print)
- ✅ CST → Walk / FindAll / FindAllComments
- ✅ Навигация: FirstChild, FirstChildOfKind, ValueNode, KeyNode, Members, Elements
- ✅ DeepEqual — полное структурное сравнение
- ✅ Debug-вывод через String()

### 4.3 Чего не хватает для production API

#### 🔴 Критично

1. **Mutation API** — нет способа изменить CST программно:
   ```go
   // Хотелось бы:
   node.SetValue("new value")
   obj.AddMember(key, value)
   arr.AppendElement(value)
   comment.SetBody("new comment")
   ```
   Без этого пакет read-only. Для JSONC-редактора/конфигуратора нужно
   собирать дерево вручную через сырые `&Node{...}`.

2. **Builder API** — нет конструкторов узлов:
   ```go
   // Хотелось бы:
   jsonc.NewString("hello")
   jsonc.NewNumber("42")
   jsonc.NewObject(members...)
   jsonc.NewArray(elements...)
   jsonc.NewLineComment(" comment text")
   jsonc.NewBlockComment(" comment text")
   ```

3. **JSON marshal/unmarshal** — нельзя использовать как `json.Marshal`:
   ```go
   // Не работает:
   data, _ := json.Marshal(doc)
   var doc jsonc.Node
   json.Unmarshal(data, &doc)
   ```

#### 🟡 Важно

4. **`Parse()` без `error`** — см. issue 1.
5. **`Node.Body()` метод** — README ссылается на `c.Body()` (строка "Comment:", `c.Body()`),
   но такого метода нет. Нужно либо добавить, либо исправить README на `c.CommentBody`.
6. **Position.IsValid()** — нет способа отличить zero-value Position от реальной позиции (0:0:0).
7. **Нет `Kind` для невалидного JSON** — `KindError` есть, но нет способа получить
   структурированную информацию об ошибке (строка, позиция, ожидаемый токен).

#### 🟢 Можно улучшить

8. **`FormatOptions`** только с `Indent` — нет `MaxInlineLength`, `SortKeys`, `Compact`.
   Для "JSONC formatter" ожидаемо больше опций.
9. **`Walk()`** — не возвращает ошибку. Нельзя прервать обход с ошибкой.
10. **Потоковый парсер** — для больших файлов нет SAX-like API.

### 4.4 Сравнение с аналогами

| Характеристика | jsonc-cst | `encoding/json` | `gopkg.in/yaml.v3` | `pkg/json` (segment) |
|---------------|-----------|-----------------|-------------------|---------------------|
| CST (с комментариями) | ✅ | ❌ AST | ❌ AST | ❌ |
| Mutation API | ❌ | ✅ | ✅ | ❌ |
| Streaming parser | ❌ | ✅ (Decoder) | ✅ (Decoder) | ✅ |
| JSON / JSONC | JSONC | JSON | YAML | JSON |
| Комментарии | ✅ сохраняет | ❌ теряет | ❌ теряет | ❌ |
| Pretty-print | ✅ кастомный | ✅ (MarshalIndent) | ❌ | ❌ |

---

## 5. Итоговая оценка

| Категория | Оценка | 
|-----------|--------|
| **Архитектура** | 🟢 9/10 — чистое разделение, CST-подход корректен |
| **Code quality** | 🟢 9/10 — 0 линт-ошибок, 22 линтера, 43% кода — тесты |
| **PBT покрытие** | 🟡 7/10 — 31 PBT тест, но пробелы в невалидных числах |
| **Production readiness** | 🟡 6/10 — нужен tag, `go 1.25`, Parse signature |
| **Публичный API** | 🟡 5/10 — read-only, нет Builder/Mutation/JSON-маршалинга |
| **Документация** | 🟢 8/10 — README хорош, но есть `Body()` vs `CommentBody` |
| **CI/CD** | 🟢 9/10 — матрица, race detector, линтер |

**Общая оценка готовности к продакшену: 7/10**

Пакет отлично подходит для **чтения и форматирования** JSONC-файлов.
Для **записи/редактирования** нужны mutation/builder API.

### Ближайшие шаги

1. **Срочно:** создать git tag (`v0.1.0`), понизить `go 1.25` до `1.22`
2. **API:** убрать `error` из `Parse()` или сделать осмысленным
3. **README:** исправить `c.Body()` на `c.CommentBody`, добавить больше примеров
4. **Parser:** улучшить `scanNumber` — отклонять невалидные последовательности
5. **Editor use-case:** добавить Builder API (NewString, NewObject, AddMember, etc.)
6. **Pre-commit:** исправить race condition в скрипте
