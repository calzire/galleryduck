(function () {
  function stopOngoingAnimation(content) {
    if (content._collapseTimer) {
      window.clearTimeout(content._collapseTimer);
      content._collapseTimer = null;
    }
    if (content._collapseOnEnd) {
      content.removeEventListener("transitionend", content._collapseOnEnd);
      content._collapseOnEnd = null;
    }
  }

  function initCollapsibles(root) {
    var scope = root || document;
    var sections = scope.querySelectorAll("[data-collapsible]");
    sections.forEach(function (section) {
      var trigger = section.querySelector("[data-collapsible-trigger]");
      var content = section.querySelector("[data-collapsible-content]");
      var icon = section.querySelector("[data-collapsible-icon]");
      var iconDown = section.querySelector("[data-collapsible-icon-down]");
      var iconUp = section.querySelector("[data-collapsible-icon-up]");
      if (!trigger || !content) return;

      content.style.overflow = "hidden";
      content.style.transition = "max-height 180ms ease, opacity 180ms ease";

      function setOpen(open, animate) {
        var shouldAnimate = animate === true;
        stopOngoingAnimation(content);
        section.dataset.collapsibleOpen = open ? "true" : "false";
        trigger.setAttribute("aria-expanded", open ? "true" : "false");

        if (open) {
          content.classList.remove("hidden");
          if (!shouldAnimate) {
            content.style.opacity = "1";
            content.style.maxHeight = "none";
          } else {
            content.style.opacity = "0";
            content.style.maxHeight = "0px";
            // Force layout so transition starts from collapsed state.
            content.offsetHeight;
            requestAnimationFrame(function () {
              content.style.opacity = "1";
              content.style.maxHeight = content.scrollHeight + "px";
            });
            content._collapseOnEnd = function (event) {
              if (event.propertyName !== "max-height") return;
              if (section.dataset.collapsibleOpen === "true") {
                content.style.maxHeight = "none";
              }
              stopOngoingAnimation(content);
            };
            content.addEventListener("transitionend", content._collapseOnEnd);
            // Fallback for interrupted transitions.
            content._collapseTimer = window.setTimeout(function () {
              if (section.dataset.collapsibleOpen === "true") {
                content.style.maxHeight = "none";
              }
              stopOngoingAnimation(content);
            }, 260);
          }
        } else {
          if (!shouldAnimate) {
            content.style.opacity = "0";
            content.style.maxHeight = "0px";
            content.classList.add("hidden");
          } else {
            if (content.style.maxHeight === "none") {
              content.style.maxHeight = content.scrollHeight + "px";
            } else {
              content.style.maxHeight = content.scrollHeight + "px";
            }
            // Force layout so close transition starts from expanded state.
            content.offsetHeight;
            content.style.maxHeight = content.scrollHeight + "px";
            content.style.opacity = "1";
            requestAnimationFrame(function () {
              content.style.maxHeight = "0px";
              content.style.opacity = "0";
            });
            content._collapseOnEnd = function (event) {
              if (event.propertyName !== "max-height") return;
              if (section.dataset.collapsibleOpen !== "true") {
                content.classList.add("hidden");
              }
              stopOngoingAnimation(content);
            };
            content.addEventListener("transitionend", content._collapseOnEnd);
            // Fallback for interrupted transitions.
            content._collapseTimer = window.setTimeout(function () {
              if (section.dataset.collapsibleOpen !== "true") {
                content.classList.add("hidden");
              }
              stopOngoingAnimation(content);
            }, 260);
          }
        }

        if (iconDown && iconUp) {
          iconDown.classList.toggle("hidden", !open);
          iconUp.classList.toggle("hidden", open);
        }
        if (icon) {
          icon.style.transform = open ? "rotate(0deg)" : "rotate(-90deg)";
        }
      }

      var initialOpen = section.dataset.collapsibleOpen !== "false";
      setOpen(initialOpen, false);

      // Guard against duplicate bindings if init is called multiple times.
      if (!trigger.dataset.collapsibleBound) {
        trigger.addEventListener("click", function () {
          var open = section.dataset.collapsibleOpen !== "true";
          setOpen(open, true);
        });
        trigger.dataset.collapsibleBound = "1";
      }
    });
  }

  function initSubtypeFilters(root) {
    var scope = root || document;
    var containers = scope.querySelectorAll("[data-subtype-filter]");
    containers.forEach(function (container) {
      var master = container.querySelector("input[name='__subtypes_all__']");
      var items = container.querySelectorAll("input[name='sub_type']");
      if (!master || !items.length) {
        if (master) {
          master.checked = false;
          master.indeterminate = false;
          master.disabled = true;
        }
        return;
      }

      function syncMasterFromItems() {
        var checkedCount = 0;
        items.forEach(function (item) {
          if (item.checked) checkedCount += 1;
        });
        master.disabled = false;
        master.checked = checkedCount === items.length;
        master.indeterminate = checkedCount > 0 && checkedCount < items.length;
      }

      function setAllItems(checked) {
        items.forEach(function (item) {
          item.checked = checked;
        });
        syncMasterFromItems();
      }

      syncMasterFromItems();

      if (!master.dataset.subtypesBound) {
        master.addEventListener("change", function () {
          setAllItems(master.checked);
        });
        master.dataset.subtypesBound = "1";
      }

      items.forEach(function (item) {
        if (item.dataset.subtypeBound) return;
        item.addEventListener("change", function () {
          syncMasterFromItems();
        });
        item.dataset.subtypeBound = "1";
      });
    });
  }

  function initThemeToggles(root) {
    var scope = root || document;
    var buttons = scope.querySelectorAll(".js-theme-toggle");
    buttons.forEach(function (button) {
      if (button.dataset.themeBound) return;

      function resolveTheme(theme) {
        if (theme === "dark" || theme === "light") return theme;
        var prefersDark =
          window.matchMedia &&
          window.matchMedia("(prefers-color-scheme: dark)").matches;
        return prefersDark ? "dark" : "light";
      }

      function currentThemePref() {
        return (
          localStorage.getItem("galleryduck-theme") ||
          document.documentElement.dataset.theme ||
          "system"
        );
      }

      function applyTheme(theme) {
        var resolved = resolveTheme(theme);
        document.documentElement.dataset.theme = theme;
        document.documentElement.dataset.themeResolved = resolved;
        document.documentElement.classList.toggle("dark", resolved === "dark");
        if (document.body) {
          document.body.classList.toggle("theme-dark", resolved === "dark");
          document.body.dataset.themeResolved = resolved;
        }
        return resolved;
      }

      function setButtonState(prefOverride) {
        var pref = prefOverride || currentThemePref();
        var resolved = applyTheme(pref);
        var icon = button.querySelector("img");
        var iconForMode =
          resolved === "dark" ? "/assets/svg/moon.svg" : "/assets/svg/sun.svg";
        if (icon) {
          icon.src = iconForMode;
        }
        var nextMode = resolved === "dark" ? "light" : "dark";
        button.setAttribute("aria-label", "Switch to " + nextMode + " mode");
        button.setAttribute("title", "Switch to " + nextMode + " mode");
      }

      async function persistTheme(theme) {
        try {
          await fetch("/api/theme", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ theme: theme }),
          });
        } catch (_) {
          // Ignore network errors; local persistence still works.
        }
      }

      button.addEventListener("click", function () {
        var pref = currentThemePref();
        var next = resolveTheme(pref) === "dark" ? "light" : "dark";
        try {
          localStorage.setItem("galleryduck-theme", next);
        } catch (_) {
          // Ignore storage errors (e.g., restricted mode); runtime theme still applies.
        }
        applyTheme(next);
        setButtonState(next);
        persistTheme(next);
      });

      setButtonState();
      button.dataset.themeBound = "1";
    });
  }

  function initGalleryShortcuts(root) {
    var scope = root || document;
    var gallery = scope.querySelector("[data-gallery-root]");
    if (!gallery || gallery.dataset.shortcutsBound) return;

    function isTypingContext(el) {
      if (!el) return false;
      var tag = (el.tagName || "").toLowerCase();
      return tag === "input" || tag === "textarea" || tag === "select" || el.isContentEditable;
    }

    function onKeyDown(event) {
      if (isTypingContext(document.activeElement)) return;
      if (document.querySelector("[data-modal]:not(.hidden)")) return;
      if (event.key === "/") {
        event.preventDefault();
        var search = document.querySelector("input[name='search']");
        if (search) search.focus();
        return;
      }
      if (event.key === "ArrowRight") {
        var nextBtn = document.querySelector("[data-page-next]");
        if (nextBtn && !nextBtn.disabled) nextBtn.click();
      } else if (event.key === "ArrowLeft") {
        var prevBtn = document.querySelector("[data-page-prev]");
        if (prevBtn && !prevBtn.disabled) prevBtn.click();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    gallery.dataset.shortcutsBound = "1";
  }

  function initFilterEnterSubmit(root) {
    var scope = root || document;
    var forms = scope.querySelectorAll("form#filters-form");
    forms.forEach(function (form) {
      if (form.dataset.enterSubmitBound) return;
      form.addEventListener("keydown", function (event) {
        if (event.key !== "Enter") return;
        var target = event.target;
        if (!target) return;
        var tag = (target.tagName || "").toLowerCase();
        if (tag === "textarea") return;
        event.preventDefault();
        if (typeof form.requestSubmit === "function") {
          form.requestSubmit();
        } else {
          form.submit();
        }
      });
      form.dataset.enterSubmitBound = "1";
    });
  }

  function initMediaCardPreview(root) {
    var scope = root || document;
    var modal = document.getElementById("gallery-media-preview-modal");
    if (!modal) return;
    var panel = modal.querySelector("[data-modal-panel]");
    var closeBtn = document.getElementById("media-preview-close");
    var prevBtn = document.getElementById("media-preview-prev");
    var nextBtn = document.getElementById("media-preview-next");
    var fullscreenBtn = document.getElementById("media-preview-fullscreen");
    var titleEl = document.getElementById("media-preview-title");
    var metaEl = document.getElementById("media-preview-meta");
    var imageEl = document.getElementById("media-preview-image");
    var videoEl = document.getElementById("media-preview-video");
    var audioEl = document.getElementById("media-preview-audio");
    if (!panel || !closeBtn || !prevBtn || !nextBtn || !fullscreenBtn || !titleEl || !metaEl || !imageEl || !videoEl || !audioEl) {
      return;
    }

    function getCards() {
      return Array.prototype.slice.call(document.querySelectorAll("[data-media-card]"));
    }

    function setCurrentIndex(index) {
      modal.dataset.previewIndex = String(index);
    }

    function getCurrentIndex() {
      var n = parseInt(modal.dataset.previewIndex || "-1", 10);
      return Number.isNaN(n) ? -1 : n;
    }

    function updateNavState(index, total) {
      prevBtn.disabled = index <= 0;
      nextBtn.disabled = index >= total - 1;
    }

    function getActiveMediaElement() {
      if (!imageEl.classList.contains("hidden")) return imageEl;
      if (!videoEl.classList.contains("hidden")) return videoEl;
      if (!audioEl.classList.contains("hidden")) return audioEl;
      return null;
    }

    function hideAllPlayers() {
      imageEl.classList.add("hidden");
      videoEl.classList.add("hidden");
      audioEl.classList.add("hidden");
      videoEl.pause();
      audioEl.pause();
      videoEl.removeAttribute("src");
      audioEl.removeAttribute("src");
      imageEl.removeAttribute("src");
    }

    function openModalAt(index) {
      var cards = getCards();
      if (!cards.length) return;
      if (index < 0) index = 0;
      if (index >= cards.length) index = cards.length - 1;
      var card = cards[index];
      if (!card) return;

      var mediaType = card.dataset.mediaType || "";
      var mediaSrc = card.dataset.mediaSrc || "";
      var mediaName = card.dataset.mediaName || "Media";
      var mediaSubType = card.dataset.mediaSubtype || "";
      var mediaDate = card.dataset.mediaDate || "";
      if (!mediaSrc) return;

      titleEl.textContent = mediaName;
      metaEl.textContent = [mediaType, mediaSubType, mediaDate].filter(Boolean).join(" · ");
      hideAllPlayers();

      if (mediaType === "image") {
        imageEl.src = mediaSrc;
        imageEl.alt = mediaName;
        imageEl.classList.remove("hidden");
      } else if (mediaType === "video") {
        videoEl.src = mediaSrc;
        videoEl.classList.remove("hidden");
        videoEl.play().catch(function () {});
      } else if (mediaType === "audio") {
        audioEl.src = mediaSrc;
        audioEl.classList.remove("hidden");
        audioEl.play().catch(function () {});
      } else {
        return;
      }

      setCurrentIndex(index);
      updateNavState(index, cards.length);
      modal.classList.remove("hidden");
      document.body.classList.add("overflow-hidden");
    }

    function openModalFromCard(card) {
      var cards = getCards();
      var index = cards.indexOf(card);
      if (index < 0) return;
      openModalAt(index);
    }

    function showPrev() {
      var index = getCurrentIndex();
      if (index <= 0) return;
      openModalAt(index - 1);
    }

    function showNext() {
      var index = getCurrentIndex();
      var total = getCards().length;
      if (index < 0 || index >= total - 1) return;
      openModalAt(index + 1);
    }

    async function toggleFullscreen() {
      var element = getActiveMediaElement();
      if (!element) return;
      if (document.fullscreenElement) {
        await document.exitFullscreen();
      } else if (element.requestFullscreen) {
        await element.requestFullscreen();
      }
    }

    function closeModal() {
      modal.classList.add("hidden");
      document.body.classList.remove("overflow-hidden");
      hideAllPlayers();
      setCurrentIndex(-1);
    }

    if (!modal.dataset.previewModalBound) {
      closeBtn.addEventListener("click", closeModal);
      prevBtn.addEventListener("click", showPrev);
      nextBtn.addEventListener("click", showNext);
      fullscreenBtn.addEventListener("click", function () {
        toggleFullscreen().catch(function () {});
      });
      modal.addEventListener("click", function (event) {
        if (event.target === modal) {
          closeModal();
        }
      });
      document.addEventListener("keydown", function (event) {
        if (modal.classList.contains("hidden")) return;
        if (event.key === "Escape") {
          closeModal();
          return;
        }
        if (event.key === "ArrowLeft") {
          event.preventDefault();
          showPrev();
          return;
        }
        if (event.key === "ArrowRight") {
          event.preventDefault();
          showNext();
        }
      });
      modal.dataset.previewModalBound = "1";
    }

    var cards = scope.querySelectorAll("[data-media-card]");
    cards.forEach(function (card) {
      if (card.dataset.previewCardBound) return;
      card.addEventListener("click", function () {
        openModalFromCard(card);
      });
      card.addEventListener("keydown", function (event) {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        openModalFromCard(card);
      });
      card.dataset.previewCardBound = "1";
    });
  }

  window.GalleryDuckUI = window.GalleryDuckUI || {};
  window.GalleryDuckUI.initCollapsibles = initCollapsibles;
  window.GalleryDuckUI.initSubtypeFilters = initSubtypeFilters;
  window.GalleryDuckUI.initThemeToggles = initThemeToggles;
  window.GalleryDuckUI.initGalleryShortcuts = initGalleryShortcuts;
  window.GalleryDuckUI.initFilterEnterSubmit = initFilterEnterSubmit;
  window.GalleryDuckUI.initMediaCardPreview = initMediaCardPreview;
})();
