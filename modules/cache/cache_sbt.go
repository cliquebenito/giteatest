package cache

import (
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
	"fmt"
)

// Item объект кеширования
type Item struct {
	Status int
	Body   interface{}
}

// toJson сериализует в JSON
func (item Item) toJson() (string, error) {
	bytes, err := json.Marshal(item)
	return string(bytes), err
}

// toItem десериализует в Item
func toItem(inputValue interface{}) (Item, error) {
	var stringValue string
	if value, ok := inputValue.(string); ok {
		stringValue = value
	} else {
		if stringer, ok := inputValue.(fmt.Stringer); ok {
			stringValue = stringer.String()
		} else {
			stringValue = fmt.Sprintf("%s", value)
		}
	}

	//десериализуем
	var item = &Item{}
	err := json.Unmarshal([]byte(stringValue), item)

	if err == nil {
		return Item{
				Status: item.Status,
				Body:   item.Body},
			nil
	} else {
		return Item{}, err
	}
}

// GetItem возвращает значение по ключу, либо вызывает функцию, если значение не найдено
func GetItem(key string, getFunc func() (Item, error)) (Item, error) {
	//если соединение отсутствует или настройка указана как 0 или явно указано не кешировать JSON - не кешировать, то просто возвращаем результат выполнения функции
	if conn == nil || setting.CacheService.TTL == 0 || !setting.CacheService.Json {
		return getFunc()
	}
	//достаем из кеша
	cached := conn.Get(key)

	//ничего не найдено, вычисляем значение с помощью функции
	if cached == nil {
		return calculate(key, getFunc)
	}
	//найдено
	item, err := toItem(cached)
	//не получилось десериализовать, вычисляем значение с помощью функции
	if err != nil {
		value, err2 := calculate(key, getFunc)
		if err2 != nil {
			//вычислили с ошибкой
			return value, err2
		}
		//вычислили без ошибки, докидываем ошибку десерилизации
		return value, &CanNotBeCached{
			Key:    key,
			Reason: err,
		}
	}
	return item, nil
}

// Calculate вычисляем значение с помощью функции и кладем его в кеш
func calculate(key string, getFunc func() (Item, error)) (Item, error) {
	value, err := getFunc()
	if err != nil {
		return value, err
	}

	//если функция нам вернула ошибочный статус, то не записываем в кеш
	if value.Status >= 400 {
		return value, nil
	}

	jsonValue, err := value.toJson()

	//не смогли сериализовать для добавления в кеш
	if err != nil {
		return value, &CanNotBeCached{
			Key:    key,
			Reason: err,
		}
	}

	err = conn.Put(key, jsonValue, setting.CacheService.TTLSeconds())

	// проблемы с записью в БД кеша
	if err != nil {
		return value, &CanNotBeCached{
			Key:    key,
			Reason: err,
		}
	}

	return value, nil
}

// RemoveItem удаляет значение Item из кеша, если включено кеширование JSON
func RemoveItem(key string) {
	if !setting.CacheService.Json {
		return
	}
	Remove(key)
}

// CanNotBeCached ошибка, если значение не может быть закешировано
type CanNotBeCached struct {
	Key    string
	Reason error
}

func (e CanNotBeCached) Error() string {
	return fmt.Sprintf("value for key %s can not be cached, reason: %v", e.Key, e.Reason)
}

func IsCanNotBeCached(err error) bool {
	_, ok := err.(CanNotBeCached)
	return ok
}
