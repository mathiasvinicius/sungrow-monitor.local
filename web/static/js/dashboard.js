// Dashboard JavaScript - Sungrow Monitor

const API_BASE = '/api/v1';
const UPDATE_INTERVAL = 5000; // 5 seconds

// DOM Elements
const elements = {
    statusDot: document.getElementById('status-dot'),
    statusText: document.getElementById('status-text'),
    powerValue: document.getElementById('power-value'),
    powerBar: document.getElementById('power-bar'),
    dailyEnergy: document.getElementById('daily-energy'),
    totalEnergy: document.getElementById('total-energy'),
    mppt1Voltage: document.getElementById('mppt1-voltage'),
    mppt1Current: document.getElementById('mppt1-current'),
    mppt1Power: document.getElementById('mppt1-power'),
    mppt2Voltage: document.getElementById('mppt2-voltage'),
    mppt2Current: document.getElementById('mppt2-current'),
    mppt2Power: document.getElementById('mppt2-power'),
    gridVoltage: document.getElementById('grid-voltage'),
    gridFrequency: document.getElementById('grid-frequency'),
    gridCurrent: document.getElementById('grid-current'),
    powerFactor: document.getElementById('power-factor'),
    runningState: document.getElementById('running-state'),
    temperature: document.getElementById('temperature'),
    serialNumber: document.getElementById('serial-number'),
    lastUpdate: document.getElementById('last-update'),
    insightStatus: document.getElementById('insight-status'),
    insightDetail: document.getElementById('insight-detail'),
    insightWeather: document.getElementById('insight-weather'),
    insightWeatherIcon: document.getElementById('insight-weather-icon'),
    insightWeatherText: document.getElementById('insight-weather-text'),
    insightComparison: document.getElementById('insight-comparison')
};

const panelOpacityInput = document.getElementById('panel-opacity');
const panelOpacityValue = document.getElementById('panel-opacity-value');
const PANEL_OPACITY_KEY = 'panelOpacity';
const PANEL_OPACITY_VERSION_KEY = 'panelOpacityVersion';
const PANEL_OPACITY_VERSION = '2';
const PANEL_OPACITY_MIN = 0;
const PANEL_OPACITY_MAX = 100;

// Nominal power in Watts
const NOMINAL_POWER = 5000;
const INSIGHT_INTERVAL = 60000; // 60 seconds

function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
}

function loadPanelTransparency() {
    let storedValue = null;
    try {
        const raw = localStorage.getItem(PANEL_OPACITY_KEY);
        const parsed = Number(raw);
        if (Number.isFinite(parsed)) {
            storedValue = parsed;
        }
    } catch (error) {
        return null;
    }

    if (storedValue === null) {
        return null;
    }

    try {
        const version = localStorage.getItem(PANEL_OPACITY_VERSION_KEY);
        if (version !== PANEL_OPACITY_VERSION) {
            const migrated = 100 - storedValue;
            localStorage.setItem(PANEL_OPACITY_KEY, String(migrated));
            localStorage.setItem(PANEL_OPACITY_VERSION_KEY, PANEL_OPACITY_VERSION);
            return migrated;
        }
    } catch (error) {
        return storedValue;
    }

    return storedValue;
}

function applyPanelOpacity(value) {
    const percent = clamp(Math.round(value), PANEL_OPACITY_MIN, PANEL_OPACITY_MAX);
    const alpha = ((100 - percent) / 100).toFixed(2);
    document.documentElement.style.setProperty('--card-bg-alpha', alpha);
    if (panelOpacityValue) {
        panelOpacityValue.textContent = `${percent}%`;
    }
    if (panelOpacityInput) {
        panelOpacityInput.value = String(percent);
    }
    try {
        localStorage.setItem(PANEL_OPACITY_KEY, String(percent));
        localStorage.setItem(PANEL_OPACITY_VERSION_KEY, PANEL_OPACITY_VERSION);
    } catch (error) {
        console.warn('Falha ao salvar transparência:', error);
    }
}

// Fetch data from API
async function fetchStatus() {
    try {
        const response = await fetch(`${API_BASE}/status`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        updateDashboard(data);
        setOnlineStatus(data.is_online === true);
    } catch (error) {
        console.error('Error fetching status:', error);
        setOnlineStatus(false);
    }
}

async function fetchInsights() {
    try {
        const response = await fetch(`${API_BASE}/insights/production`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        updateInsights(data);
    } catch (error) {
        console.error('Error fetching insights:', error);
        setInsightsFallback();
    }
}

// Update dashboard with data
function updateDashboard(data) {
    // Power
    const power = data.total_active_power_w || 0;
    elements.powerValue.textContent = formatNumber(power);
    const powerPercent = Math.min((power / NOMINAL_POWER) * 100, 100);
    elements.powerBar.style.width = `${powerPercent}%`;

    // Energy
    elements.dailyEnergy.textContent = formatNumber(data.daily_energy_kwh, 1);
    elements.totalEnergy.textContent = formatNumber(data.total_energy_kwh, 1);

    // MPPT 1
    elements.mppt1Voltage.textContent = formatNumber(data.mppt1_voltage_v, 1);
    elements.mppt1Current.textContent = formatNumber(data.mppt1_current_a, 2);
    const mppt1Power = (data.mppt1_voltage_v || 0) * (data.mppt1_current_a || 0);
    elements.mppt1Power.textContent = formatNumber(mppt1Power, 0);

    // MPPT 2
    elements.mppt2Voltage.textContent = formatNumber(data.mppt2_voltage_v, 1);
    elements.mppt2Current.textContent = formatNumber(data.mppt2_current_a, 2);
    const mppt2Power = (data.mppt2_voltage_v || 0) * (data.mppt2_current_a || 0);
    elements.mppt2Power.textContent = formatNumber(mppt2Power, 0);

    // Grid
    elements.gridVoltage.textContent = formatNumber(data.grid_voltage_v, 1);
    elements.gridFrequency.textContent = formatNumber(data.grid_frequency_hz, 1);
    elements.gridCurrent.textContent = formatNumber(data.grid_current_a, 2);
    elements.powerFactor.textContent = formatNumber(data.power_factor, 3);

    // Status
    elements.runningState.textContent = data.running_state_string || '--';
    elements.temperature.textContent = formatNumber(data.temperature_c, 1);
    elements.serialNumber.textContent = data.serial_number || '--';

    // Last update
    if (data.timestamp) {
        const date = new Date(data.timestamp);
        elements.lastUpdate.textContent = date.toLocaleString('pt-BR');
    }
}

function updateInsights(data) {
    elements.insightStatus.textContent = data.message || '--';
    elements.insightDetail.textContent = formatInsightDetail(data);
    const weatherText = formatInsightWeather(data);
    if (elements.insightWeatherText) {
        elements.insightWeatherText.textContent = weatherText || '--';
    } else {
        elements.insightWeather.textContent = weatherText || '--';
    }
    renderWeatherIcon(weatherText);

    if (elements.insightComparison) {
        elements.insightComparison.title = formatComparisonTooltip(data);
    }

    if (data.expected_avg_w && data.actual_power_w && data.ratio) {
        const ratioPercent = (data.ratio * 100).toFixed(0);
        elements.insightComparison.textContent = `${data.actual_power_w}W / ${Math.round(data.expected_avg_w)}W (${ratioPercent}%)`;
    } else if (data.expected_avg_w) {
        elements.insightComparison.textContent = `Média: ${Math.round(data.expected_avg_w)}W`;
    } else {
        elements.insightComparison.textContent = '--';
    }
}

function formatInsightWeather(data) {
    if (!data || !data.weather_label) {
        return '--';
    }
    const label = capitalizeFirst(String(data.weather_label));
    return label || '--';
}

function normalizeWeatherLabel(value) {
    return String(value || '')
        .toLowerCase()
        .normalize('NFD')
        .replace(/[\u0300-\u036f]/g, '')
        .trim();
}

function getWeatherIconKey(label) {
    const normalized = normalizeWeatherLabel(label);
    if (!normalized || normalized === '--') {
        return '';
    }
    if (normalized.includes('temporal') || normalized.includes('trovoada')) {
        return 'storm';
    }
    if (normalized.includes('chuva forte')) {
        return 'heavy-rain';
    }
    if (normalized.includes('chuva')) {
        return 'rain';
    }
    if (normalized.includes('nevoeiro') || normalized.includes('neblina') || normalized.includes('fog')) {
        return 'fog';
    }
    if (normalized.includes('poucas nuvens') || normalized.includes('parcialmente')) {
        return 'partly-cloudy';
    }
    if (normalized.includes('encoberto') || normalized.includes('nublado')) {
        return 'cloudy';
    }
    if (normalized.includes('limpo') || normalized.includes('clear')) {
        return 'clear';
    }
    return 'cloudy';
}

function renderWeatherIcon(label) {
    if (!elements.insightWeatherIcon) {
        return;
    }

    const iconKey = getWeatherIconKey(label);
    if (!iconKey) {
        elements.insightWeatherIcon.innerHTML = '';
        return;
    }

    const icons = {
        clear: `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="3.5"></circle>
                <path d="M12 2v2.5M12 19.5V22M4.2 4.2l1.8 1.8M18 18l1.8 1.8M2 12h2.5M19.5 12H22M4.2 19.8l1.8-1.8M18 6l1.8-1.8"></path>
            </svg>
        `,
        'partly-cloudy': `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="8.5" cy="8.5" r="3"></circle>
                <path d="M8.5 2.5v1.8M8.5 12.2V14M3.4 3.4l1.3 1.3M12.3 12.3l1.3 1.3"></path>
                <path d="M6 18h9a3.5 3.5 0 0 0 0-7 4.5 4.5 0 0 0-8.7-1.2A3.5 3.5 0 0 0 6 18z"></path>
            </svg>
        `,
        cloudy: `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 16a4 4 0 0 0 4 4h8a4 4 0 0 0 0-8 5 5 0 0 0-9.7-1.2A4 4 0 0 0 4 16z"></path>
            </svg>
        `,
        rain: `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 14a4 4 0 0 0 4 4h7a4 4 0 0 0 0-8 5 5 0 0 0-9.7-1.2A4 4 0 0 0 4 14z"></path>
                <path d="M8 20l-1 2M12 20l-1 2M16 20l-1 2"></path>
            </svg>
        `,
        'heavy-rain': `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 13a4 4 0 0 0 4 4h7a4 4 0 0 0 0-8 5 5 0 0 0-9.7-1.2A4 4 0 0 0 4 13z"></path>
                <path d="M7 19l-1.2 2.5M11 19l-1.2 2.5M15 19l-1.2 2.5M19 19l-1.2 2.5"></path>
            </svg>
        `,
        storm: `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 14a4 4 0 0 0 4 4h7a4 4 0 0 0 0-8 5 5 0 0 0-9.7-1.2A4 4 0 0 0 4 14z"></path>
                <path d="M12 16l-2 4h2l-1.5 4 4-6h-2l1.5-2z"></path>
            </svg>
        `,
        fog: `
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 14a4 4 0 0 0 4 4h7a4 4 0 0 0 0-8 5 5 0 0 0-9.7-1.2A4 4 0 0 0 4 14z"></path>
                <path d="M3 19h10M2 22h14"></path>
            </svg>
        `
    };

    elements.insightWeatherIcon.innerHTML = icons[iconKey] || '';
}

function capitalizeFirst(value) {
    const text = (value || '').trim();
    if (!text) {
        return '';
    }
    return text.charAt(0).toUpperCase() + text.slice(1);
}

function setInsightsFallback() {
    elements.insightStatus.textContent = '--';
    elements.insightDetail.textContent = '--';
    if (elements.insightWeatherText) {
        elements.insightWeatherText.textContent = '--';
    } else {
        elements.insightWeather.textContent = '--';
    }
    elements.insightComparison.textContent = '--';
    if (elements.insightComparison) {
        elements.insightComparison.title = '';
    }
    if (elements.insightWeatherIcon) {
        elements.insightWeatherIcon.innerHTML = '';
    }
}

function formatInsightDetail(data) {
    if (!data || !data.status) {
        return '--';
    }

    if (data.status === 'night') {
        return 'Janela noturna';
    }
    if (data.status === 'insufficient_history') {
        return 'Histórico insuficiente';
    }
    if (data.status === 'low_power_unexpected') {
        return 'Possível anomalia';
    }
    if (data.status === 'low_power_weather') {
        return 'Impacto do clima';
    }
    if (data.status === 'normal') {
        return 'Dentro do esperado';
    }
    return '--';
}

function formatComparisonTooltip(data) {
    if (!data) {
        return '';
    }
    const days = data.window_days ? `${data.window_days} dias` : '';
    const bucket = data.bucket_minutes ? `${data.bucket_minutes} min` : '';
    const suffix = [days, bucket].filter(Boolean).join(', ');
    if (suffix) {
        return `Comparação com a média histórica da mesma faixa horária (${suffix}).`;
    }
    return 'Comparação com a média histórica da mesma faixa horária.';
}

// Set online/offline status
function setOnlineStatus(online) {
    if (online) {
        elements.statusDot.classList.remove('offline');
        elements.statusDot.classList.add('online');
        elements.statusText.textContent = 'Online';
    } else {
        elements.statusDot.classList.remove('online');
        elements.statusDot.classList.add('offline');
        elements.statusText.textContent = 'Offline';
    }
}

// Format number with decimals
function formatNumber(value, decimals = 0) {
    if (value === null || value === undefined || isNaN(value)) {
        return '--';
    }
    return Number(value).toFixed(decimals);
}

// Initial fetch
fetchStatus();
fetchInsights();

// Panel opacity control
if (panelOpacityInput) {
    let initialValue = 70;
    const stored = loadPanelTransparency();
    if (Number.isFinite(stored)) {
        initialValue = stored;
    }
    applyPanelOpacity(initialValue);
    panelOpacityInput.addEventListener('input', (event) => {
        applyPanelOpacity(Number(event.target.value));
    });
}

// Set up interval for updates
setInterval(fetchStatus, UPDATE_INTERVAL);
setInterval(fetchInsights, INSIGHT_INTERVAL);

// Health check
async function checkHealth() {
    try {
        const response = await fetch('/health');
        const data = await response.json();
        console.log('Health check:', data);
    } catch (error) {
        console.error('Health check failed:', error);
    }
}

// Check health on load
checkHealth();
