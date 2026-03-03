import { useState, useEffect, useRef } from 'react'

const API = 'http://localhost:8080'

const DNS_LIST = [
  { label: '1.1.1.1 Cloudflare', value: '1.1.1.1' },
  { label: '8.8.8.8 Google', value: '8.8.8.8' },
  { label: '9.9.9.9 Quad9', value: '9.9.9.9' },
  { label: '77.88.8.8 Яндекс', value: '77.88.8.8' },
  { label: '94.140.14.14 AdGuard', value: '94.140.14.14' },
]

export default function App() {
  const [server, setServer] = useState('')
  const [logs, setLogs] = useState([])
  const [stealing, setStealing] = useState(false)
  const [auth, setAuth] = useState(null)
  const [history, setHistory] = useState([])
  const [activeDns, setActiveDns] = useState('1.1.1.1')
  const [toast, setToast] = useState(null)
  const [tab, setTab] = useState('steal')
  const logRef = useRef(null)
  const pollRef = useRef(null)

  useEffect(() => {
    checkAuth()
    loadHistory()
  }, [])

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight
  }, [logs])

  const showToast = (msg, type = 'info') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const checkAuth = async () => {
    try {
      const r = await fetch(`${API}/api/auth`)
      const d = await r.json()
      setAuth(d.authenticated)
    } catch { setAuth(false) }
  }

  const loadHistory = async () => {
    try {
      const r = await fetch(`${API}/api/history`)
      const d = await r.json()
      setHistory(d || [])
    } catch {}
  }

  const pollLogs = () => {
    pollRef.current = setInterval(async () => {
      try {
        const r = await fetch(`${API}/api/logs`)
        const d = await r.json()
        setLogs(d || [])
        const last = d?.[d.length - 1]
        if (last && (last.message.includes('Готово') || last.message.includes('ошибка подключения'))) {
          clearInterval(pollRef.current)
          setStealing(false)
          loadHistory()
          if (last.message.includes('Готово')) showToast('Паки сохранены!', 'success')
          else showToast('Ошибка подключения', 'error')
        }
      } catch {}
    }, 500)
  }

  const stealPacks = async () => {
    if (!server.trim()) { showToast('Укажи IP сервера', 'error'); return }
    setStealing(true)
    setLogs([])
    try {
      await fetch(`${API}/api/steal`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ server: server.trim() })
      })
      pollLogs()
    } catch {
      showToast('Сервер недоступен', 'error')
      setStealing(false)
    }
  }

  const setDNS = async (dns) => {
    setActiveDns(dns)
    try {
      await fetch(`${API}/api/dns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dns })
      })
      showToast('DNS: ' + dns, 'success')
    } catch { showToast('Ошибка смены DNS', 'error') }
  }

  const clearLogs = async () => {
    await fetch(`${API}/api/logs/clear`, { method: 'POST' })
    setLogs([])
  }

  const s = styles

  return (
    <div style={s.root}>
      <div style={s.bg} />
      <div style={s.grid} />

      <div style={s.container}>
        {/* Шапка */}
        <div style={s.header}>
          <div style={s.logo}>PackSteal</div>
          <div style={{ ...s.badge, ...(auth ? s.badgeOk : {}) }}>
            <div style={{ ...s.dot, ...(auth ? s.dotOk : {}) }} />
            {auth === null ? 'Проверка...' : auth ? 'Xbox авторизован' : 'Не авторизован'}
          </div>
        </div>

        {/* Табы */}
        <div style={s.tabs}>
          {['steal', 'history', 'settings'].map(t => (
            <button key={t} style={{ ...s.tab, ...(tab === t ? s.tabActive : {}) }} onClick={() => setTab(t)}>
              {t === 'steal' ? '⚡ Steal' : t === 'history' ? '📦 История' : '⚙ Настройки'}
            </button>
          ))}
        </div>

        {tab === 'steal' && <>
          {/* Инпут */}
          <div style={s.card}>
            <div style={s.cardTitle}>Сервер</div>
            <div style={s.row}>
              <input
                style={s.input}
                value={server}
                onChange={e => setServer(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && stealPacks()}
                placeholder="zeqa.net:19132"
              />
              <button
                style={{ ...s.btn, ...(stealing ? s.btnLoading : {}) }}
                onClick={stealPacks}
                disabled={stealing}
              >
                {stealing ? '⏳' : '▶'}
              </button>
            </div>
          </div>

          {/* Лог */}
          <div style={s.card}>
            <div style={{ ...s.cardTitle, display: 'flex', justifyContent: 'space-between' }}>
              <span>Консоль</span>
              <span style={{ cursor: 'pointer', fontSize: 10 }} onClick={clearLogs}>очистить</span>
            </div>
            <div ref={logRef} style={s.logBox}>
              {logs.length === 0
                ? <div style={s.empty}>Лог пуст</div>
                : logs.map((l, i) => (
                  <div key={i} style={s.logLine}>
                    <span style={s.logTime}>{l.time}</span>
                    <span style={{ ...s.logMsg, ...(l.type === 'success' ? s.logSuccess : l.type === 'error' ? s.logError : {}) }}>
                      {l.message}
                    </span>
                  </div>
                ))
              }
            </div>
          </div>
        </>}

        {tab === 'history' && (
          <div style={s.card}>
            <div style={{ ...s.cardTitle, display: 'flex', justifyContent: 'space-between' }}>
              <span>Скачанные паки</span>
              <span style={{ cursor: 'pointer', fontSize: 10 }} onClick={loadHistory}>обновить</span>
            </div>
            {history.length === 0
              ? <div style={s.empty}>Нет скачанных паков</div>
              : <div style={s.histGrid}>
                {history.map((h, i) => (
                  <div key={i} style={s.histItem} onClick={() => { setServer(h.server.replace(/_(\d+)$/, ':$1')); setTab('steal') }}>
                    <div style={s.histServer}>{h.server.replace(/_/g, '.')}</div>
                    <div style={s.histCount}>{h.packs} паков</div>
                  </div>
                ))}
              </div>
            }
          </div>
        )}

        {tab === 'settings' && (
          <div style={s.card}>
            <div style={s.cardTitle}>DNS сервер</div>
            <div style={s.dnsRow}>
              {DNS_LIST.map(d => (
                <button
                  key={d.value}
                  style={{ ...s.chip, ...(activeDns === d.value ? s.chipActive : {}) }}
                  onClick={() => setDNS(d.value)}
                >
                  {d.label}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {toast && (
        <div style={{ ...s.toast, ...(toast.type === 'success' ? s.toastOk : toast.type === 'error' ? s.toastErr : {}) }}>
          {toast.msg}
        </div>
      )}
    </div>
  )
}

const styles = {
  root: { minHeight: '100vh', background: '#0a0a0c', color: '#d4dde6', fontFamily: "'JetBrains Mono', monospace", overflow: 'hidden', position: 'relative' },
  bg: { position: 'fixed', top: -200, left: -200, width: 600, height: 600, background: 'radial-gradient(circle, rgba(107,143,168,0.08) 0%, transparent 70%)', pointerEvents: 'none', zIndex: 0 },
  grid: { position: 'fixed', inset: 0, backgroundImage: 'linear-gradient(rgba(168,184,200,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(168,184,200,0.03) 1px, transparent 1px)', backgroundSize: '40px 40px', pointerEvents: 'none', zIndex: 0 },
  container: { position: 'relative', zIndex: 1, maxWidth: 600, margin: '0 auto', padding: '32px 16px' },
  header: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 28 },
  logo: { fontFamily: 'sans-serif', fontWeight: 800, fontSize: 26, letterSpacing: -1, background: 'linear-gradient(135deg, #d4dde6, #6b8fa8)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' },
  badge: { display: 'flex', alignItems: 'center', gap: 8, padding: '5px 12px', background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 20, fontSize: 11, color: '#5a6a7a' },
  badgeOk: { color: '#d4dde6' },
  dot: { width: 6, height: 6, borderRadius: '50%', background: '#5a6a7a' },
  dotOk: { background: '#4a9a6a' },
  tabs: { display: 'flex', gap: 8, marginBottom: 16 },
  tab: { padding: '8px 16px', background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 8, color: '#5a6a7a', fontSize: 11, cursor: 'pointer', fontFamily: 'monospace' },
  tabActive: { background: 'rgba(107,143,168,0.15)', border: '1px solid rgba(107,143,168,0.3)', color: '#a8b8c8' },
  card: { background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 14, padding: 20, marginBottom: 14 },
  cardTitle: { fontSize: 10, fontWeight: 600, letterSpacing: 2, textTransform: 'uppercase', color: '#5a6a7a', marginBottom: 14 },
  row: { display: 'flex', gap: 10 },
  input: { flex: 1, background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 8, padding: '10px 14px', color: '#d4dde6', fontFamily: 'monospace', fontSize: 13, outline: 'none' },
  btn: { padding: '10px 18px', background: 'rgba(107,143,168,0.2)', border: '1px solid rgba(107,143,168,0.4)', borderRadius: 8, color: '#a8b8c8', fontSize: 14, cursor: 'pointer', fontFamily: 'monospace' },
  btnLoading: { opacity: 0.5, cursor: 'not-allowed' },
  logBox: { height: 260, overflowY: 'auto', fontSize: 11, lineHeight: 1.8 },
  logLine: { display: 'flex', gap: 10, paddingBottom: 2, borderBottom: '1px solid rgba(255,255,255,0.02)' },
  logTime: { color: '#5a6a7a', minWidth: 60 },
  logMsg: { color: '#d4dde6' },
  logSuccess: { color: '#6abf8a' },
  logError: { color: '#bf6a6a' },
  empty: { color: '#5a6a7a', fontSize: 12, textAlign: 'center', padding: 20 },
  histGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 10 },
  histItem: { background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 8, padding: 12, cursor: 'pointer' },
  histServer: { fontSize: 11, color: '#a8b8c8', marginBottom: 4, wordBreak: 'break-all' },
  histCount: { fontSize: 10, color: '#5a6a7a' },
  dnsRow: { display: 'flex', flexWrap: 'wrap', gap: 8 },
  chip: { padding: '6px 12px', background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 20, fontSize: 11, color: '#5a6a7a', cursor: 'pointer', fontFamily: 'monospace' },
  chipActive: { background: 'rgba(107,143,168,0.1)', border: '1px solid rgba(107,143,168,0.4)', color: '#a8b8c8' },
  toast: { position: 'fixed', bottom: 24, right: 24, padding: '10px 18px', background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 10, fontSize: 12, zIndex: 100 },
  toastOk: { borderColor: 'rgba(74,154,106,0.4)', color: '#6abf8a' },
  toastErr: { borderColor: 'rgba(154,74,74,0.4)', color: '#bf6a6a' },
}
