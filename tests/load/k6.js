import { check, sleep } from 'k6';
import crypto from 'k6/crypto';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const successRate = new Counter('success');
const errorRate = new Counter('errors');
const latency = new Trend('latency', true);

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '20s', target: 50 },
    { duration: '10s', target: 100 },
    { duration: '20s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'],
    'http_req_failed': ['rate<0.01'],
  },
  maxRedirects: 0,
};
const PORT = __ENV.PORT
const BASE_URL = `http://localhost:${PORT}`;
const SECRET = __ENV.WEBHOOK_SECRET
const cpf = '52998224725';

export default function () {
  const callId = `CH-K6-${__VU}-${__ITER}-${Date.now()}`;
  
  const payload = JSON.stringify({
    chamado_id: callId,
    tipo: 'status_change',
    cpf,
    status_anterior: 'em_analise',
    status_novo: 'em_execucao',
    titulo: 'Teste de Carga K6',
    descricao: 'Notificação gerada durante teste de carga com k6',
    timestamp: new Date().toISOString(),
  });


  const signature = crypto.hmac('sha256', SECRET, payload, 'hex');

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Signature-256': `sha256=${signature}`,
    },
  };

  const res = http.post(`${BASE_URL}/webhook`, payload, params);

  latency.add(res.timings.duration);

  if (res.status === 201 || res.status === 200) {
    successRate.add(1);
  } else {
    errorRate.add(1);
  }

  check(res, {
    'status 201 ou 200': (r) => r.status === 201 || r.status === 200,
    'tempo < 200ms': (r) => r.timings.duration < 200,
  });

  sleep(0.1);
}

export function handleSummary(data) {
  return {
    'stdout': `
╔══════════════════════════════════════════════════════════════╗
║              RESULTADOS DO TESTE DE CARGA (K6)               ║
╚══════════════════════════════════════════════════════════════╝

  Requisições total: ${data.metrics.http_reqs.values.count}
  Duração:           ${data.state.testRunDurationMs}ms
  Requisições/s:     ${data.metrics.http_reqs.values.rate.toFixed(1)}
  Latência média:    ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms
  Latência P95:      ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
  Latência P99:      ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms
`,
  };
}