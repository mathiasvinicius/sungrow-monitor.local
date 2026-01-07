// Settings JavaScript - Sungrow Monitor

const API_BASE = '/api/v1';

// DOM Elements
const form = document.getElementById('settings-form');
const testBtn = document.getElementById('test-btn');
const saveBtn = document.getElementById('save-btn');
const alertContainer = document.getElementById('alert-container');
const weatherForm = document.getElementById('weather-form');
const weatherSaveBtn = document.getElementById('weather-save-btn');

// Current config elements
const currentIP = document.getElementById('current-ip');
const currentPort = document.getElementById('current-port');
const currentSlaveID = document.getElementById('current-slave-id');
const currentTimeout = document.getElementById('current-timeout');
const currentWeatherEnabled = document.getElementById('current-weather-enabled');
const currentWeatherProvider = document.getElementById('current-weather-provider');
const currentWeatherLocation = document.getElementById('current-weather-location');
const currentBgProvider = document.getElementById('current-bg-provider');
const currentUnsplashKey = document.getElementById('current-unsplash-key');

// Form inputs
const ipInput = document.getElementById('inverter-ip');
const portInput = document.getElementById('inverter-port');
const slaveIDInput = document.getElementById('slave-id');
const timeoutInput = document.getElementById('timeout');
const weatherEnabled = document.getElementById('weather-enabled');
const weatherProvider = document.getElementById('weather-provider');
const weatherAPIKey = document.getElementById('weather-api-key');
const weatherCity = document.getElementById('weather-city');
const weatherCountry = document.getElementById('weather-country');
const weatherLatitude = document.getElementById('weather-latitude');
const weatherLongitude = document.getElementById('weather-longitude');
const weatherUnits = document.getElementById('weather-units');
const backgroundForm = document.getElementById('background-form');
const backgroundSaveBtn = document.getElementById('background-save-btn');
const unsplashAccessKey = document.getElementById('unsplash-access-key');
const unsplashClearKey = document.getElementById('unsplash-clear-key');

// Load current configuration on page load
async function loadCurrentConfig() {
    try {
        const response = await fetch(`${API_BASE}/config/inverter`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const config = await response.json();

        // Populate form
        ipInput.value = config.ip;
        portInput.value = config.port;
        slaveIDInput.value = config.slave_id;
        timeoutInput.value = config.timeout_seconds;

        // Update display
        updateCurrentConfigDisplay(config);

    } catch (error) {
        console.error('Failed to load config:', error);
        showAlert('error', 'Erro ao carregar configuração atual');
    }
}

// Update current configuration display
function updateCurrentConfigDisplay(config) {
    currentIP.textContent = config.ip;
    currentPort.textContent = config.port;
    currentSlaveID.textContent = config.slave_id;
    currentTimeout.textContent = `${config.timeout_seconds}s`;
}

async function loadWeatherConfig() {
    try {
        const response = await fetch(`${API_BASE}/config/weather`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const config = await response.json();

        weatherEnabled.checked = !!config.enabled;
        weatherProvider.value = normalizeWeatherProvider(config.provider) || 'openweather';
        weatherAPIKey.value = config.api_key || '';
        weatherCity.value = config.city || '';
        weatherCountry.value = config.country || '';
        weatherLatitude.value = config.latitude || '';
        weatherLongitude.value = config.longitude || '';
        weatherUnits.value = config.units || 'metric';

        updateWeatherConfigDisplay(config);
    } catch (error) {
        console.error('Failed to load weather config:', error);
        showAlert('error', 'Erro ao carregar configuração de clima');
    }
}

function updateWeatherConfigDisplay(config) {
    currentWeatherEnabled.textContent = config.enabled ? 'Sim' : 'Não';
    currentWeatherProvider.textContent = formatWeatherProvider(config.provider) || '--';
    const location = formatWeatherLocation(config);
    currentWeatherLocation.textContent = location || '--';
}

async function loadBackgroundConfig() {
    try {
        const response = await fetch(`${API_BASE}/config/background`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const config = await response.json();

        updateBackgroundConfigDisplay(config);
    } catch (error) {
        console.error('Failed to load background config:', error);
        showAlert('error', 'Erro ao carregar configuração do fundo');
    }
}

function updateBackgroundConfigDisplay(config) {
    if (!config) {
        currentBgProvider.textContent = '--';
        currentUnsplashKey.textContent = '--';
        return;
    }

    const provider = (config.provider || '').toLowerCase();
    if (provider === 'unsplash') {
        currentBgProvider.textContent = 'Unsplash';
    } else if (provider === 'bing') {
        currentBgProvider.textContent = 'Bing';
    } else {
        currentBgProvider.textContent = provider || '--';
    }

    currentUnsplashKey.textContent = config.has_unsplash_key ? 'Configurada' : 'Não configurada';
}

function normalizeWeatherProvider(value) {
    const provider = (value || '').toLowerCase();
    if (provider === 'open-meteo' || provider === 'open_meteo') {
        return 'openmeteo';
    }
    return provider;
}

function formatWeatherProvider(value) {
    const provider = normalizeWeatherProvider(value);
    if (provider === 'openmeteo') {
        return 'Open-Meteo';
    }
    if (provider === 'openweather') {
        return 'OpenWeather';
    }
    return provider;
}

// Test connection before saving
testBtn.addEventListener('click', async () => {
    // Validate form first
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }

    const config = {
        ip: ipInput.value.trim(),
        port: parseInt(portInput.value),
        slave_id: parseInt(slaveIDInput.value),
        timeout_seconds: parseInt(timeoutInput.value)
    };

    setButtonLoading(testBtn, true, 'Testando...');

    try {
        const response = await fetch(`${API_BASE}/config/inverter/test`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        const result = await response.json();

        if (response.ok && result.success) {
            showAlert('success', `Conexão bem-sucedida! Serial: ${result.serial_number || 'N/A'}`);
        } else {
            showAlert('error', `Falha no teste: ${result.error || 'Erro desconhecido'}`);
        }
    } catch (error) {
        console.error('Test failed:', error);
        showAlert('error', 'Erro ao testar conexão: ' + error.message);
    } finally {
        setButtonLoading(testBtn, false, 'Testar Conexão');
    }
});

// Save and apply configuration
form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const config = {
        ip: ipInput.value.trim(),
        port: parseInt(portInput.value),
        slave_id: parseInt(slaveIDInput.value),
        timeout_seconds: parseInt(timeoutInput.value)
    };

    setButtonLoading(saveBtn, true, 'Salvando...');

    try {
        const response = await fetch(`${API_BASE}/config/inverter`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        const result = await response.json();

    if (response.ok) {
            showAlert('success', 'Configuração salva e aplicada com sucesso!');
            updateCurrentConfigDisplay(config);

            // Reload config to confirm
            setTimeout(() => loadCurrentConfig(), 1000);
        } else {
            showAlert('error', `Erro ao salvar: ${result.error || 'Erro desconhecido'}`);
        }
    } catch (error) {
        console.error('Save failed:', error);
        showAlert('error', 'Erro ao salvar configuração: ' + error.message);
    } finally {
        setButtonLoading(saveBtn, false, 'Salvar e Aplicar');
    }
});

weatherForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const config = {
        enabled: !!weatherEnabled.checked,
        provider: weatherProvider.value,
        api_key: weatherAPIKey.value.trim(),
        city: weatherCity.value.trim(),
        country: weatherCountry.value.trim(),
        latitude: parseFloat(weatherLatitude.value) || 0,
        longitude: parseFloat(weatherLongitude.value) || 0,
        units: weatherUnits.value
    };

    setButtonLoading(weatherSaveBtn, true, 'Salvando...');

    try {
        const response = await fetch(`${API_BASE}/config/weather`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        const result = await response.json();

        if (response.ok) {
            showAlert('success', 'Configuração de clima salva com sucesso!');
            updateWeatherConfigDisplay(config);
            setTimeout(() => loadWeatherConfig(), 1000);
        } else {
            showAlert('error', `Erro ao salvar clima: ${result.error || 'Erro desconhecido'}`);
        }
    } catch (error) {
        console.error('Weather save failed:', error);
        showAlert('error', 'Erro ao salvar clima: ' + error.message);
    } finally {
        setButtonLoading(weatherSaveBtn, false, 'Salvar Clima');
    }
});

backgroundForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const payload = {
        clear_unsplash_key: !!unsplashClearKey.checked
    };

    const keyValue = (unsplashAccessKey.value || '').trim();
    if (keyValue) {
        payload.unsplash_access_key = keyValue;
    }

    setButtonLoading(backgroundSaveBtn, true, 'Salvando...');

    try {
        const response = await fetch(`${API_BASE}/config/background`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const result = await response.json();

        if (response.ok) {
            showAlert('success', 'Configuração de fundo salva com sucesso!');
            updateBackgroundConfigDisplay(result);
            unsplashAccessKey.value = '';
            unsplashClearKey.checked = false;
            setTimeout(() => loadBackgroundConfig(), 1000);
        } else {
            showAlert('error', `Erro ao salvar fundo: ${result.error || 'Erro desconhecido'}`);
        }
    } catch (error) {
        console.error('Background save failed:', error);
        showAlert('error', 'Erro ao salvar fundo: ' + error.message);
    } finally {
        setButtonLoading(backgroundSaveBtn, false, 'Salvar Fundo');
    }
});

// Show alert message
function showAlert(type, message) {
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;

    alertContainer.innerHTML = '';
    alertContainer.appendChild(alert);

    // Auto-dismiss after 5 seconds
    setTimeout(() => {
        alert.style.opacity = '0';
        alert.style.transition = 'opacity 0.3s ease';
        setTimeout(() => alert.remove(), 300);
    }, 5000);
}

// Set button loading state
function setButtonLoading(button, loading, text) {
    if (loading) {
        button.disabled = true;
        button.querySelector('span').textContent = text;
    } else {
        button.disabled = false;
        button.querySelector('span').textContent = text;
    }
}

function formatWeatherLocation(config) {
    if (config.city) {
        return config.country ? `${config.city}, ${config.country}` : config.city;
    }
    if (config.latitude && config.longitude) {
        return `${Number(config.latitude).toFixed(4)}, ${Number(config.longitude).toFixed(4)}`;
    }
    return '';
}

// Initialize on page load
loadCurrentConfig();
loadWeatherConfig();
loadBackgroundConfig();

// Update status indicator
async function updateStatus() {
    try {
        const response = await fetch(`${API_BASE}/status`);
        const data = await response.json();

        const statusDot = document.getElementById('status-dot');
        const statusText = document.getElementById('status-text');

        if (data.is_online) {
            statusDot.classList.remove('offline');
            statusDot.classList.add('online');
            statusText.textContent = 'Online';
        } else {
            statusDot.classList.remove('online');
            statusDot.classList.add('offline');
            statusText.textContent = 'Offline';
        }
    } catch (error) {
        console.error('Status update failed:', error);
    }
}

// Update status every 5 seconds
updateStatus();
setInterval(updateStatus, 5000);
