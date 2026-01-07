(() => {
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
        fetchBackground(wallpaper);
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();
