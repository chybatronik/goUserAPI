 #!/bin/bash
  # test-post-users.sh

  echo "🧪 Тестирование POST /users endpoint..."

  echo "1. ✅ Создание валидного пользователя:"
  curl -X POST http://localhost:8080/users \
    -H "Content-Type: application/json" \
    -d '{
      "first_name": "Test",
      "last_name": "User",
      "age": 30
    }' \
    -w "\n📝 Status: %{http_code}\n" \
    -s | tail -n 1

  echo "2. ❌ Пустое имя (ожидается 400):"
  curl -X POST http://localhost:8080/users \
    -H "Content-Type: application/json" \
    -d '{
      "first_name": "",
      "last_name": "User",
      "age": 30
    }' \
    -w "\n📝 Status: %{http_code}\n" \
    -s | tail -n 1

  echo "3. ❌ Неверный возраст (ожидается 400):"
  curl -X POST http://localhost:8080/users \
    -H "Content-Type: application/json" \
    -d '{
      "first_name": "Test",
      "last_name": "User",
      "age": 150
    }' \
    -w "\n📝 Status: %{http_code}\n" \
    -s | tail -n 1

  echo "4. ❌ GET метод (ожидается 405):"
  curl -X GET http://localhost:8080/users \
    -w "\n📝 Status: %{http_code}\n" \
    -s | tail -n 1

  echo "✅ Тестирование завершено!"
