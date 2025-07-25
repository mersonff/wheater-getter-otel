# Sistema de Temperatura por CEP

Este projeto é composto por dois serviços em Go que, juntos, recebem um CEP, identificam a cidade e retornam o clima atual (Celsius, Fahrenheit, Kelvin), com tracing distribuído via OpenTelemetry e Zipkin.

## Como rodar

1. **Clone o repositório e configure o ambiente:**
```bash
git clone <repository-url>
cd weather-getter-otel
cp env.example .env # Edite o .env e coloque sua chave do WeatherAPI
```

2. **Suba os serviços:**
```bash
docker-compose up --build
```

3. **Teste a aplicação:**
```bash
./test.sh
```

4. **Acesse o Zipkin para ver os traces:**
- http://localhost:9411

## Endpoints principais

### Service A (porta 8080)
- `POST /cep` — Recebe `{ "cep": "29902555" }` e retorna cidade e temperaturas
- `GET /health` — Health check

### Service B (porta 8081)
- `POST /weather` — Usado internamente pelo Service A
- `GET /health` — Health check

## Exemplo de uso

```bash
curl -X POST http://localhost:8080/cep \
  -H "Content-Type: application/json" \
  -d '{"cep": "29902555"}'
```

Resposta esperada:
```json
{
  "city": "Vitória",
  "temp_C": 25.5,
  "temp_F": 77.9,
  "temp_K": 298.65
}
```

## Erros comuns
- `422` — CEP inválido: `{ "message": "invalid zipcode" }`
- `404` — CEP não encontrado: `{ "message": "can not find zipcode" }`

## Observabilidade
- Todos os requests são traceados com OTEL e enviados para o Zipkin.
- Veja o fluxo completo de cada requisição em http://localhost:9411

## Variáveis de ambiente principais
- `WEATHER_API_KEY` — Chave da WeatherAPI (obrigatória)
- `PORT` — Porta do serviço (8080 ou 8081)
- `ZIPKIN_URL` — URL do Zipkin (default já funciona com o compose)

## Dúvidas?
- Veja os logs: `docker-compose logs`
- Teste: `./test.sh`
- Veja o Zipkin: http://localhost:9411
