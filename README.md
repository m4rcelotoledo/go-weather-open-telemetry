# 🌤️ Go Weather com OpenTelemetry

Sistema de microsserviços em Go que implementa consulta de clima por CEP com **tracing distribuído** usando OpenTelemetry e Zipkin.

## 🏗️ Arquitetura

O sistema é composto por **2 microsserviços** que se comunicam para fornecer informações de clima baseadas em CEP:

- **Service A (Porta 8080)**: Recebe requisições, valida CEP e encaminha para Service B
- **Service B (Porta 8081)**: Orquestra chamadas para APIs externas (ViaCEP + WeatherAPI)

## 🚀 Como Executar

### 1. Configurar API Key
Crie um arquivo `.env` na pasta `service-b/`:
```bash
cd service-b
echo "WEATHER_API_KEY=sua_chave_da_weatherapi_aqui" > .env
```

**⚠️ IMPORTANTE:** Você precisa de uma chave gratuita da [WeatherAPI](https://www.weatherapi.com/) para que o sistema funcione.

### 2. Iniciar a Stack Completa
```bash
docker-compose up --build -d
```

### 3. Verificar Status
```bash
docker-compose ps
```

## 🧪 Testando o Sistema

### Teste Básico
```bash
# Consultar clima por CEP
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "29902555"}'
```

### Resposta Esperada
```json
{
  "city": "Linhares",
  "temp_C": 20.4,
  "temp_F": 68.7,
  "temp_K": 293.5
}
```

### Teste de Validação
```bash
# CEP inválido
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "123"}'
```

## 🏗️ **ARQUITETURA DO SISTEMA**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Service A     │    │   Service B     │    │   Prometheus    │
│  (CEP Handler)  │◄──►│  (Orchestrator) │◄──►│   (Métricas)    │
│   Porta 8080    │    │   Porta 8081    │    │   Porta 9090    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Zipkin UI      │    │  OTEL Collector │    │  Métricas       │
│  Porta 9411     │    │  Porta 4318     │    │  Performance    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🔍 Observabilidade

### Zipkin (Visualização de Traces)
- **URL**: http://localhost:9411
- **Função**: Visualizar traces distribuídos entre os serviços
- **Prometheus**: Coleta e armazena métricas de performance

### OpenTelemetry Collector
- **Porta**: 4318
- **Função**: Coletar e processar telemetria dos serviços

### Prometheus
- **Porta**: 9090
- **URL**: http://localhost:9090
- **Função**: Coleta métricas de performance dos serviços

## 📊 Estrutura do Projeto

```
/
├── service-a/                 # Serviço de entrada (porta 8080)
├── service-b/                 # Serviço de orquestração (porta 8081)
├── docker-compose.yml         # Orquestração dos serviços
├── otel-collector-config.yaml # Configuração do OTEL Collector
└── README.md                  # Este arquivo
```

## 🧪 Testes de Integração

Execute o script de testes para verificar toda a stack:
```bash
chmod +x run_integration_tests.sh
./run_integration_tests.sh
```

## 🐳 Serviços Docker

| Serviço | Porta | Função |
|---------|-------|--------|
| **Service A** | 8080 | API de entrada e validação |
| **Service B** | 8081 | Orquestração de APIs externas |
| **OTEL Collector** | 4318 | Coleta de telemetria |
| **Zipkin** | 9411 | Visualização de traces |

## 🔧 Desenvolvimento

### Recompilar Serviços
```bash
# Reconstruir e reiniciar
docker-compose down
docker-compose up --build -d
```

### Logs dos Serviços
```bash
# Ver logs de um serviço específico
docker-compose logs service-a
docker-compose logs service-b
```

## 📈 Funcionalidades

- ✅ **Validação de CEP** (8 dígitos numéricos)
- ✅ **Integração ViaCEP** para busca de cidade
- ✅ **Integração WeatherAPI** para dados climáticos
- ✅ **Cálculos automáticos** (Fahrenheit e Kelvin)
- ✅ **Tracing distribuído** com OpenTelemetry
- ✅ **Visualização de traces** no Zipkin
- ✅ **Tratamento de erros** robusto
- ✅ **Health checks** para todos os serviços

## 🎯 Tecnologias Utilizadas

- **Go 1.24** - Linguagem principal
- **OpenTelemetry** - Observabilidade e tracing
- **Zipkin** - Visualização de traces
- **Docker Compose** - Orquestração de containers
- **ViaCEP API** - Busca de endereços por CEP
- **WeatherAPI** - Dados meteorológicos
