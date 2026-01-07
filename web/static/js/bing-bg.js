(() => {
    const PANEL_OPACITY_KEY = 'panelOpacity';
    const PANEL_OPACITY_VERSION_KEY = 'panelOpacityVersion';
    const PANEL_OPACITY_VERSION = '2';
    const PANEL_OPACITY_MIN = 0;
    const PANEL_OPACITY_MAX = 100;

    const applyStoredOpacity = () => {
        try {
            const raw = localStorage.getItem(PANEL_OPACITY_KEY);
            let stored = Number(raw);
            if (!Number.isFinite(stored)) {
                return;
            }
            const version = localStorage.getItem(PANEL_OPACITY_VERSION_KEY);
            if (version !== PANEL_OPACITY_VERSION) {
                stored = 100 - stored;
                localStorage.setItem(PANEL_OPACITY_KEY, String(stored));
                localStorage.setItem(PANEL_OPACITY_VERSION_KEY, PANEL_OPACITY_VERSION);
            }
            const clamped = Math.min(Math.max(Math.round(stored), PANEL_OPACITY_MIN), PANEL_OPACITY_MAX);
            const alpha = ((100 - clamped) / 100).toFixed(2);
            document.documentElement.style.setProperty('--card-bg-alpha', alpha);
        } catch (error) {
            console.warn('Falha ao aplicar transparência:', error);
        }
    };

    const fetchBackground = async (wallpaper) => {
        try {
            const response = await fetch("/api/v1/background/wallpaper?mkt=pt-BR");
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            const data = await response.json();
            if (!data || !data.url) {
                throw new Error("Resposta invalida do fundo");
            }
            wallpaper.style.backgroundImage = `url("${data.url}")`;
            wallpaper.classList.add("visible");
            const credit = data.credit || data.copyright || data.title;
            if (credit) {
                wallpaper.title = credit;
            }
        } catch (error) {
            console.error("Falha ao carregar fundo:", error);
            try {
                const response = await fetch("/api/v1/bing-wallpaper?mkt=pt-BR");
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
                const data = await response.json();
                if (!data || !data.url) {
                    throw new Error("Resposta invalida do Bing");
                }
                wallpaper.style.backgroundImage = `url("${data.url}")`;
                wallpaper.classList.add("visible");
                if (data.copyright) {
                    wallpaper.title = data.copyright;
                }
            } catch (fallbackError) {
                console.error("Fallback Bing falhou:", fallbackError);
            }
        }
    };

    const init = () => {
        const wallpaper = document.getElementById("bing-wallpaper");
        if (!wallpaper) {
            return;
        }
        applyStoredOpacity();
        fetchBackground(wallpaper);
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();
