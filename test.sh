#!/bin/bash

echo "🚀 Testando o sistema de temperatura por CEP com OTEL + Zipkin"
echo "=========================================================="

# Aguardar os serviços iniciarem
echo "⏳ Aguardando os serviços iniciarem..."
sleep 10

# Teste 1: CEP válido
echo ""
echo "📋 Teste 1: CEP válido (29902555)"
echo "--------------------------------"
curl -X POST http://localhost:8080/cep \
  -H "Content-Type: application/json" \
  -d '{"cep": "29902555"}' \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

# Teste 2: CEP inválido (formato)
echo ""
echo "📋 Teste 2: CEP inválido (formato)"
echo "--------------------------------"
curl -X POST http://localhost:8080/cep \
  -H "Content-Type: application/json" \
  -d '{"cep": "12345"}' \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

# Teste 3: CEP inválido (não encontrado)
echo ""
echo "📋 Teste 3: CEP inválido (não encontrado)"
echo "----------------------------------------"
curl -X POST http://localhost:8080/cep \
  -H "Content-Type: application/json" \
  -d '{"cep": "99999999"}' \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

# Teste 4: JSON inválido
echo ""
echo "📋 Teste 4: JSON inválido"
echo "------------------------"
curl -X POST http://localhost:8080/cep \
  -H "Content-Type: application/json" \
  -d '{"cep": }' \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

# Teste 5: Health check Service A
echo ""
echo "📋 Teste 5: Health check Service A"
echo "--------------------------------"
curl -X GET http://localhost:8080/health \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

# Teste 6: Health check Service B
echo ""
echo "📋 Teste 6: Health check Service B"
echo "--------------------------------"
curl -X GET http://localhost:8081/health \
  -w "\nStatus: %{http_code}\nTempo: %{time_total}s\n"

echo ""
echo "✅ Testes concluídos!"
echo ""
echo "🌐 Acesse o Zipkin em: http://localhost:9411"
echo "📊 Para visualizar os traces distribuídos" 