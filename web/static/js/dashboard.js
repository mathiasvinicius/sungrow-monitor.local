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
    insightComparison: document.getElementById('insight-comparison')
};

const panelOpacityInput = document.getElementById('panel-opacity');
const panelOpacityValue = document.getElementById('panel-opacity-value');
const PANEL_OPACITY_KEY = 'panelOpacity';

// Nominal power in Watts
const NOMINAL_POWER = 5000;
const INSIGHT_INTERVAL = 60000; // 60 seconds

function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
}

function applyPanelOpacity(value) {
    const percent = clamp(Math.round(value), 30, 90);
    const alpha = (percent / 100).toFixed(2);
    document.documentElement.style.setProperty('--card-bg-alpha', alpha);
    if (panelOpacityValue) {
        panelOpacityValue.textContent = `${percent}%`;
    }
    if (panelOpacityInput) {
        panelOpacityInput.value = String(percent);
    }
    try {
        localStorage.setItem(PANEL_OPACITY_KEY, String(percent));
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
    elements.insightWeather.textContent = formatInsightWeather(data);

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
    elements.insightWeather.textContent = '--';
    elements.insightComparison.textContent = '--';
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
    try {
        const stored = Number(localStorage.getItem(PANEL_OPACITY_KEY));
        if (Number.isFinite(stored)) {
            initialValue = stored;
        }
    } catch (error) {
        console.warn('Falha ao ler transparência:', error);
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
