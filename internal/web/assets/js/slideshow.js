(function () {
  function parseBool(value, fallback) {
    if (typeof value === "boolean") return value;
    if (value === "true" || value === "1") return true;
    if (value === "false" || value === "0") return false;
    return fallback;
  }

  function decodeBase64JSON(value) {
    if (!value) return [];
    const binary = atob(value.trim());
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    const jsonText = new TextDecoder("utf-8").decode(bytes);
    return JSON.parse(jsonText);
  }

  function normalizeSpeedSeconds(value) {
    const presets = [1, 2, 3, 5, 10];
    const parsed = Number(value);
    if (!Number.isFinite(parsed) || parsed <= 0) return 3;
    const sec = parsed > 20 ? parsed / 1000 : parsed;
    let nearest = presets[0];
    let bestDist = Math.abs(sec - presets[0]);
    for (let i = 1; i < presets.length; i++) {
      const dist = Math.abs(sec - presets[i]);
      if (dist < bestDist) {
        nearest = presets[i];
        bestDist = dist;
      }
    }
    return nearest;
  }

  function selectOptionByValue(select, value, fallbackValue) {
    if (!select) return;
    const desired = String(value);
    let idx = Array.from(select.options).findIndex(function (opt) {
      return opt.value === desired;
    });
    if (idx < 0) {
      idx = Array.from(select.options).findIndex(function (opt) {
        return opt.value === String(fallbackValue);
      });
    }
    select.selectedIndex = idx >= 0 ? idx : 0;
  }

  function createSlideshowPlayer(root, payload) {
    if (!root) return null;

    if (root._slideshowCleanup) {
      root._slideshowCleanup();
      root._slideshowCleanup = null;
    }

    const stageWrap = root.querySelector("[data-role='stage-wrap']");
    const stage = root.querySelector("[data-role='stage']");
    const caption = root.querySelector("[data-role='caption']");
    const progress = root.querySelector("[data-role='progress']");
    const scope = root.querySelector("[data-role='scope']");
    const empty = root.querySelector("[data-role='empty']");
    const btnPrev = root.querySelector("[data-role='prev']");
    const btnNext = root.querySelector("[data-role='next']");
    const btnPlay = root.querySelector("[data-role='play']");
    const btnClose = root.querySelector("[data-role='close']");
    const btnFullscreen = root.querySelector("[id$='-fullscreen']");
    const speedSel = root.querySelector("[data-role='speed']");
    const transitionSel = root.querySelector("[data-role='transition']");
    const randomCb = root.querySelector("input[id$='-random']");
    const loopCb = root.querySelector("input[id$='-loop']");

    const dataset = root.dataset || {};
    const data = payload || {};
    const items = Array.isArray(data.items)
      ? data.items
      : decodeBase64JSON(dataset.itemsB64 || "");
    if (scope && (data.scope_text || data.scopeText)) {
      scope.textContent = data.scope_text || data.scopeText;
    }

    const speedValue =
      data.speedMS != null ? data.speedMS : data.speed_ms != null ? data.speed_ms : dataset.speedMs;
    if (speedSel) {
      selectOptionByValue(speedSel, normalizeSpeedSeconds(speedValue), 3);
    }
    if (transitionSel && data.transition) transitionSel.value = data.transition;
    if (randomCb) randomCb.checked = parseBool(data.random, parseBool(dataset.random, false));
    if (loopCb) loopCb.checked = parseBool(data.loop, parseBool(dataset.loop, true));

    let order = items.map((_, i) => i);
    let index = 0;
    let playing = parseBool(data.autoplay, parseBool(dataset.autoplay, false));
    const autoFullscreen = parseBool(
      data.fullscreen,
      parseBool(dataset.autoFullscreen, false)
    );
    let timer = null;

    function isFullscreenActive() {
      return (
        document.fullscreenElement === stageWrap ||
        document.webkitFullscreenElement === stageWrap
      );
    }

    function updateFullscreenButton() {
      if (!btnFullscreen) return;
      const active = isFullscreenActive();
      btnFullscreen.setAttribute("aria-label", active ? "Exit fullscreen" : "Enter fullscreen");
      btnFullscreen.setAttribute("title", active ? "Exit fullscreen" : "Enter fullscreen");
    }

    async function toggleFullscreen() {
      if (!stageWrap) return;
      try {
        if (isFullscreenActive()) {
          if (document.exitFullscreen) {
            await document.exitFullscreen();
          } else if (document.webkitExitFullscreen) {
            document.webkitExitFullscreen();
          }
        } else if (stageWrap.requestFullscreen) {
          await stageWrap.requestFullscreen();
        } else if (stageWrap.webkitRequestFullscreen) {
          stageWrap.webkitRequestFullscreen();
        }
      } catch (_) {
        // Ignore browser policy/user-gesture restrictions.
      }
      updateFullscreenButton();
    }

    function shuffle(list) {
      for (let i = list.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        const tmp = list[i];
        list[i] = list[j];
        list[j] = tmp;
      }
    }

    function resetOrder() {
      order = items.map((_, i) => i);
      if (randomCb && randomCb.checked) shuffle(order);
      index = 0;
    }

    function buildMedia(item) {
      const fullscreen = isFullscreenActive();
      if (item.type === "image") {
        const img = document.createElement("img");
        img.src = item.src;
        img.alt = item.name;
        img.className = fullscreen
          ? "h-full w-full max-h-screen max-w-screen object-contain"
          : "max-h-[52vh] w-auto max-w-full object-contain rounded-lg";
        return img;
      }
      if (item.type === "video") {
        const video = document.createElement("video");
        video.src = item.src;
        video.controls = true;
        video.preload = "metadata";
        video.className = fullscreen
          ? "h-full w-full max-h-screen max-w-screen object-contain bg-black"
          : "max-h-[52vh] w-auto max-w-full rounded-lg bg-black";
        return video;
      }
      const audioWrap = document.createElement("div");
      audioWrap.className = fullscreen
        ? "w-full max-w-3xl rounded-xl border border-slate-700 bg-slate-900/90 p-5 text-slate-100"
        : "w-full max-w-xl rounded-xl border border-slate-200 bg-white p-5";
      const title = document.createElement("h3");
      title.className = fullscreen ? "font-semibold text-slate-100" : "font-semibold text-slate-900";
      title.textContent = item.name;
      const audio = document.createElement("audio");
      audio.src = item.src;
      audio.controls = true;
      audio.preload = "metadata";
      audio.className = "mt-3 w-full";
      audioWrap.appendChild(title);
      audioWrap.appendChild(audio);
      return audioWrap;
    }

    function applyTransition(el) {
      if (!transitionSel || transitionSel.value === "fade") {
        el.style.opacity = "0";
        el.style.transition = "opacity 220ms ease";
        requestAnimationFrame(() => {
          el.style.opacity = "1";
        });
        return;
      }
      el.style.transform = "translateX(18px)";
      el.style.opacity = "0";
      el.style.transition = "transform 220ms ease, opacity 220ms ease";
      requestAnimationFrame(() => {
        el.style.transform = "translateX(0)";
        el.style.opacity = "1";
      });
    }

    function updateProgress() {
      if (!progress) return;
      if (!order.length) {
        progress.textContent = "0 / 0";
        return;
      }
      progress.textContent = `${index + 1} / ${order.length}`;
    }

    function render() {
      if (!stage || !caption) return;
      stage.innerHTML = "";
      if (!order.length) {
        if (empty) empty.classList.remove("hidden");
        caption.textContent = "";
        updateProgress();
        return;
      }
      if (empty) empty.classList.add("hidden");
      const item = items[order[index]];
      const media = buildMedia(item);
      applyTransition(media);
      stage.appendChild(media);
      caption.textContent = `${item.name} · ${item.type}/${item.sub_type} · ${item.date}`;
      updateProgress();
    }

    function next() {
      if (!order.length) return;
      if (index + 1 < order.length) {
        index += 1;
      } else if (loopCb && loopCb.checked) {
        index = 0;
      } else {
        stop();
        return;
      }
      render();
    }

    function prev() {
      if (!order.length) return;
      if (index > 0) {
        index -= 1;
      } else if (loopCb && loopCb.checked) {
        index = order.length - 1;
      }
      render();
    }

    function intervalMS() {
      const parsedSec = parseInt(speedSel ? speedSel.value : "3", 10);
      return normalizeSpeedSeconds(parsedSec) * 1000;
    }

    function stopTimer() {
      if (timer !== null) {
        window.clearInterval(timer);
        timer = null;
      }
    }

    function start() {
      playing = true;
      if (btnPlay) btnPlay.textContent = "Pause";
      stopTimer();
      timer = window.setInterval(next, intervalMS());
    }

    function stop() {
      playing = false;
      if (btnPlay) btnPlay.textContent = "Play";
      stopTimer();
    }

    function togglePlay() {
      if (playing) stop();
      else start();
    }

    function onKeydown(event) {
      if (event.key === "ArrowRight") next();
      else if (event.key === "ArrowLeft") prev();
      else if (event.key === " ") {
        event.preventDefault();
        togglePlay();
      } else if (event.key.toLowerCase() === "f") {
        toggleFullscreen();
      } else if (event.key === "Escape" && btnClose) {
        btnClose.click();
      }
    }

    function onClose() {
      stop();
      if (isFullscreenActive()) {
        toggleFullscreen();
      }
      root.dispatchEvent(new CustomEvent("slideshow:close", { bubbles: true }));
    }

    let touchStartX = null;
    let touchStartY = null;
    function onTouchStart(event) {
      if (!event.touches || event.touches.length === 0) return;
      touchStartX = event.touches[0].clientX;
      touchStartY = event.touches[0].clientY;
    }
    function onTouchEnd(event) {
      if (touchStartX == null || !event.changedTouches || event.changedTouches.length === 0) return;
      const dx = event.changedTouches[0].clientX - touchStartX;
      const dy = event.changedTouches[0].clientY - touchStartY;
      touchStartX = null;
      touchStartY = null;
      if (Math.abs(dx) < 40 || Math.abs(dx) < Math.abs(dy)) return;
      if (dx < 0) next();
      else prev();
    }

    if (btnPrev) btnPrev.addEventListener("click", prev);
    if (btnNext) btnNext.addEventListener("click", next);
    if (btnPlay) btnPlay.addEventListener("click", togglePlay);
    if (btnFullscreen) btnFullscreen.addEventListener("click", toggleFullscreen);
    if (btnClose) btnClose.addEventListener("click", onClose);
    if (speedSel) {
      speedSel.addEventListener("change", function () {
        if (playing) start();
      });
    }
    if (transitionSel) transitionSel.addEventListener("change", render);
    if (randomCb) {
      randomCb.addEventListener("change", function () {
        resetOrder();
        render();
      });
    }
    if (loopCb) loopCb.addEventListener("change", function () {});

    document.addEventListener("keydown", onKeydown);
    if (stage) {
      stage.addEventListener("touchstart", onTouchStart, { passive: true });
      stage.addEventListener("touchend", onTouchEnd, { passive: true });
    }
    document.addEventListener("fullscreenchange", render);
    document.addEventListener("webkitfullscreenchange", render);
    document.addEventListener("fullscreenchange", updateFullscreenButton);
    document.addEventListener("webkitfullscreenchange", updateFullscreenButton);

    resetOrder();
    render();
    updateFullscreenButton();

    if (autoFullscreen && !isFullscreenActive()) {
      toggleFullscreen();
    }

    if (playing) start();
    else stop();

    root._slideshowCleanup = function () {
      stopTimer();
      document.removeEventListener("keydown", onKeydown);
      if (stage) {
        stage.removeEventListener("touchstart", onTouchStart);
        stage.removeEventListener("touchend", onTouchEnd);
      }
      document.removeEventListener("fullscreenchange", render);
      document.removeEventListener("webkitfullscreenchange", render);
      document.removeEventListener("fullscreenchange", updateFullscreenButton);
      document.removeEventListener("webkitfullscreenchange", updateFullscreenButton);
    };

    return {
      next: next,
      prev: prev,
      play: start,
      pause: stop,
      toggleFullscreen: toggleFullscreen,
    };
  }

  function initSlideshowPlayers(root) {
    const scope = root || document;
    const players = scope.querySelectorAll("[data-slideshow-player]");
    players.forEach(function (player) {
      if (player.dataset.slideshowManual === "true") return;
      if (player.dataset.slideshowAutoBound === "1") return;
      createSlideshowPlayer(player, null);
      player.dataset.slideshowAutoBound = "1";
    });
  }

  window.GalleryDuckUI = window.GalleryDuckUI || {};
  window.GalleryDuckUI.createSlideshowPlayer = createSlideshowPlayer;
  window.GalleryDuckUI.initSlideshowPlayers = initSlideshowPlayers;
})();
