# Sungrow Monitor

Monitoramento do inversor **Sungrow SG5.0RS-S** via **Modbus TCP**, com:
- Dashboard Web (Gin)
- API HTTP (`/api/v1`)
- Armazenamento em **SQLite**
- Publicação via **MQTT** (com discovery para Home Assistant)

## Estrutura do projeto

- `cmd/sungrow-monitor/`: entrada do binário (CLI com Cobra: `serve`, `read`, `test`)
- `config/`: carregamento de configuração (Viper)
- `internal/`
  - `api/`: servidor HTTP (Gin) + rotas do dashboard/API
  - `collector/`: laço de coleta periódica
  - `inverter/`: leitura/parse de registradores do inversor
  - `modbus/`: cliente Modbus TCP
  - `mqtt/`: publisher MQTT + Home Assistant discovery
  - `storage/`: persistência em SQLite
- `web/`: templates HTML e assets estáticos do dashboard
- `docker/`: arquivos auxiliares do Mosquitto
- `docker-compose.yaml`: stack com `mosquitto` + `sungrow-monitor`
- `Dockerfile`: build multi-stage do binário e runtime

## Configuração (`config.yaml`)

O serviço lê o arquivo informado em `--config/-c`. Se não for informado, procura por `config.yaml` no diretório atual e em `/etc/sungrow-monitor/`.

Exemplo:

```yaml
inverter:
  ip: "172.16.0.120"
  port: 502
  slave_id: 1
  timeout: 10s

collector:
  interval: 30s
  enabled: true

api:
  port: 8080
  enabled: true
  web_path: "/app/web"

mqtt:
  enabled: true
  broker: "tcp://mosquitto:1883"
  topic_prefix: "sungrow"
  client_id: "sungrow-monitor"
  username: ""
  password: ""

database:
  path: "/data/sungrow.db"
```

## Como usar (Docker)

1. Ajuste o `config.yaml` (principalmente `inverter.ip`)
2. Suba os serviços:

```bash
docker compose up -d --build
```

3. Acesse:
- Dashboard: `http://<IP_DO_HOST>:8080/`
- Health: `http://<IP_DO_HOST>:8080/health`

Se o inversor estiver na rede local e houver problema de roteamento a partir da rede bridge do Docker, use `network_mode: host` (comentado no `docker-compose.yaml`).

## Como usar (Local)

Requer Go 1.22+.

```bash
go run ./cmd/sungrow-monitor serve -c ./config.yaml
```

Comandos úteis:
- `sungrow-monitor serve -c <config>`: inicia coleta + API + MQTT
- `sungrow-monitor read -c <config>`: lê uma vez e imprime JSON
- `sungrow-monitor test -c <config>`: testa conexão Modbus TCP

## API HTTP (principais rotas)

- `GET /health`: estado do serviço/coleta
- `GET /api/v1/status`: último estado lido do inversor (se disponível)
- `GET /api/v1/readings`: leituras (com `limit`, ou `from/to` em RFC3339)
- `GET /api/v1/readings/latest`: última leitura persistida
- `GET /api/v1/energy/daily?date=YYYY-MM-DD`
- `GET /api/v1/energy/total`
- `GET /api/v1/stats/daily?date=YYYY-MM-DD`

## MQTT / Home Assistant

Quando `mqtt.enabled: true`, o serviço publica:
- Tópicos de métricas em: `<topic_prefix>/SG5.0RS-S/<campo>`
- Status completo em JSON em: `<topic_prefix>/SG5.0RS-S/status`
- Discovery do Home Assistant em: `homeassistant/sensor/sungrow/<id>/config`

## Troubleshooting

- **HTTP não abre**: confirme se o container está publicando `8080:8080` e se o processo iniciou (logs: `docker logs -f sungrow-monitor`).
- **MQTT não conecta**: garanta que `mqtt.broker` aponta para um host resolvível a partir do container (no `docker-compose`, `mosquitto` funciona via rede interna).
- **Erro Modbus** (`connect: connection refused/timeout`): verifique IP/porta do inversor, conectividade de rede e se o Modbus TCP está habilitado no equipamento.
