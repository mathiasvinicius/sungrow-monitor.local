(() => {
    const DEFAULT_MARKET = "pt-BR";
    const DEFAULT_INDEX = 0;

    const normalizeLabel = (value) => {
        return String(value || "")
            .toLowerCase()
            .normalize("NFD")
            .replace(/[\u0300-\u036f]/g, "");
    };

    const pickBingIndex = (insights) => {
        const label = normalizeLabel(
            insights?.weather_label ||
            insights?.weather?.description ||
            insights?.weather?.condition
        );

        if (!label) {
            return DEFAULT_INDEX;
        }
        if (label.includes("temporal") || label.includes("trovoada")) {
            return 6;
        }
        if (label.includes("chuva forte")) {
            return 5;
        }
        if (label.includes("chuva")) {
            return 4;
        }
        if (label.includes("nevoeiro") || label.includes("neblina") || label.includes("fog")) {
            return 7;
        }
        if (label.includes("encoberto") || label.includes("nublado")) {
            return 3;
        }
        if (label.includes("poucas nuvens") || label.includes("parcialmente")) {
            return 2;
        }
        if (label.includes("limpo") || label.includes("clear")) {
            return 1;
        }
        return DEFAULT_INDEX;
    };

    const fetchInsights = async () => {
        try {
            const response = await fetch("/api/v1/insights/production");
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.warn("Nao foi possivel obter clima para o fundo:", error);
            return null;
        }
    };

    const fetchBingWallpaper = async (wallpaper, index) => {
        try {
            const response = await fetch(
                `/api/v1/bing-wallpaper?mkt=${DEFAULT_MARKET}&idx=${index}`
            );
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
        } catch (error) {
            console.error("Falha ao carregar fundo do Bing:", error);
        }
    };

    const init = () => {
        const wallpaper = document.getElementById("bing-wallpaper");
        if (!wallpaper) {
            return;
        }
        fetchInsights().then((insights) => {
            const index = pickBingIndex(insights);
            fetchBingWallpaper(wallpaper, index);
        });
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();
