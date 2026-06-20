package env

import (
    "errors"
    "fmt"
    "os"
    "reflect"
    "strconv"
    "github.com/joho/godotenv"
)

// Config загружает .env файл в окружение
func Config(path string) error {
    return godotenv.Load(path)
}

// Load маппит значения из .env в struct через tags
// v должен быть pointer to struct
// Теги:
//   env:"KEY_NAME"      - ключ в .env
//   default:"value"     - умолчание (если нет в .env)
// Если нет ни в .env, ни в default → ошибка
func Load(v interface{}) error {
    // Проверка: v должен быть pointer
    val := reflect.ValueOf(v)
    if val.Kind() != reflect.Ptr {
        return errors.New("Load: ожидался pointer, gotten " + val.Kind().String())
    }
    
    // Проверка: pointer должен указывать на struct
    if val.Elem().Kind() != reflect.Struct {
        return errors.New("Load: pointer должен указывать на struct")
    }
    
    // Load .env файл
    if err := godotenv.Load(".env"); err != nil {
        // Если .env нет — продолжают с os.Getenv (уже загружено в окружение)
        if !os.IsNotExist(err) {
            return fmt.Errorf("Load: ошибка чтения .env: %w", err)
        }
    }
    
    return loadStruct(val)
}

func loadStruct(val reflect.Value) error {
    str := val.Elem()
    typ := str.Type()
    
    var missing []string
    
    for i := 0; i < str.NumField(); i++ {
        field := typ.Field(i)
        
        // Пропускаем неэкспортированные поля
        if !field.IsExported() {
            continue
        }
        
        envKey := field.Tag.Get("env")
        if envKey == "" {
            // Поле без env tag — пропускаем
            continue
        }
        
        defaultValue := field.Tag.Get("default")
        value := os.Getenv(envKey)
        
        // Если нет в .env, используем default
        if value == "" {
            if defaultValue != "" {
                value = defaultValue
            } else {
                missing = append(missing, envKey)
                continue
            }
        }
        
        // Запись значения в поле
        fieldVal := str.Field(i)
        if err := setField(fieldVal, value, field.Type); err != nil {
            return fmt.Errorf("поле %s: %w", envKey, err)
        }
    }
    
    if len(missing) > 0 {
        return errors.New("требуемые поля без значения в .env и без default: " + fmt.Sprint(missing))
    }
    
    return nil
}

func setField(field reflect.Value, value string, typ reflect.Type) error {
    switch typ.Kind() {
    case reflect.String:
        field.SetString(value)
        return nil
    
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        intVal, err := strconv.ParseInt(value, 10, 64)
        if err != nil {
            return fmt.Errorf("нельзя преобразовать '%s' в int", value)
        }
        field.SetInt(intVal)
        return nil
    
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        uintVal, err := strconv.ParseUint(value, 10, 64)
        if err != nil {
            return fmt.Errorf("нельзя преобразовать '%s' в uint", value)
        }
        field.SetUint(uintVal)
        return nil
    
    case reflect.Bool:
        boolVal, err := strconv.ParseBool(value)
        if err != nil {
            return fmt.Errorf("нельзя преобразовать '%s' в bool", value)
        }
        field.SetBool(boolVal)
        return nil
    
    case reflect.Float32, reflect.Float64:
        floatVal, err := strconv.ParseFloat(value, 64)
        if err != nil {
            return fmt.Errorf("нельзя преобразовать '%s' в float", value)
        }
        field.SetFloat(floatVal)
        return nil
    
    default:
        return fmt.Errorf("неподдерживаемый тип: %s (можно: string, int, bool, float)", typ.Kind())
    }
}