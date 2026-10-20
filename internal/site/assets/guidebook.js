// Navegação dos guias do guidebook (docs/spec.md, seções 3.3 e 9.3).
// Sem rede: tudo vem do HTML e de window.GUIDEBOOK_DATA.
(function () {
  "use strict";

  var data = window.GUIDEBOOK_DATA || {};
  var doc = document;
  var main = doc.querySelector(".gb-main");
  var pages = Array.prototype.slice.call(doc.querySelectorAll(".gb-page"));
  var navLinks = Array.prototype.slice.call(doc.querySelectorAll(".gb-nav a"));
  var dialog = doc.querySelector(".gb-dialog");
  var videoDialog = doc.querySelector(".gb-video-dialog");
  var dialogBody = dialog.querySelector(".gb-dialog-body");
  var backButtons = Array.prototype.slice.call(doc.querySelectorAll(".gb-back"));
  var baseTitle = doc.title;

  // fold tira acento e maiúscula, um caractere por vez, para que as
  // posições no texto dobrado valham no original.
  function fold(text) {
    return Array.prototype.map.call(text, function (c) {
      var f = c.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();
      return f.length === 1 ? f : c.toLowerCase().charAt(0);
    }).join("");
  }

  // Apresentação e vídeo só existem no guia do usuário.
  function isOpen(d) {
    return Boolean(d && d.open);
  }

  // Pilha do "Voltar para onde eu estava": cada navegação por link de
  // tarefa guarda o ponto de origem; voltar desfaz uma por vez.
  var backStack = [];
  var returning = null;
  var dialogOpener = null;

  // Estado do modo apresentação; as funções ficam mais abaixo.
  var presenting = false;
  var slides = [];
  var slideIndex = 0;

  doc.documentElement.classList.add("gb-js");

  function homeHash() {
    return "#/" + data.home;
  }

  function decodeHash(hash) {
    try {
      return decodeURIComponent((hash || "").replace(/^#/, ""));
    } catch (e) {
      return "";
    }
  }

  // element devolve o elemento do conteúdo apontado pelo hash.
  function element(hash) {
    var id = decodeHash(hash);
    var el = id ? doc.getElementById(id) : null;
    return el && main.contains(el) ? el : null;
  }

  function route() {
    var el = element(location.hash) || element(homeHash());
    if (!el) {
      return;
    }
    var article = el.closest(".gb-page");
    pages.forEach(function (p) {
      p.classList.toggle("gb-current", p === article);
    });
    navLinks.forEach(function (a) {
      if (decodeHash(a.getAttribute("href")) === article.id) {
        a.setAttribute("aria-current", "page");
      } else {
        a.removeAttribute("aria-current");
      }
    });
    var heading = article.querySelector("h1");
    doc.title = heading ? heading.textContent + " — " + baseTitle : baseTitle;
    renderDiagrams(article);

    var back = returning;
    returning = null;
    if (presenting) {
      loadSlides(article);
      showSlide(back && back.slide !== null ? back.slide : slideOf(el));
    } else if (back) {
      window.scrollTo(0, back.scrollY);
    } else if (el === article) {
      window.scrollTo(0, 0);
    } else {
      el.scrollIntoView();
    }
    if (heading && !presenting) {
      heading.setAttribute("tabindex", "-1");
      heading.focus({ preventScroll: true });
    }
    backButtons.forEach(function (b) {
      b.hidden = backStack.length === 0;
    });
  }

  // section copia o conteúdo apontado para o diálogo: a página inteira
  // ou o título e o que vem depois dele, até o próximo título do mesmo
  // nível ou acima.
  function section(el) {
    var frag = doc.createDocumentFragment();
    if (el.classList.contains("gb-page")) {
      Array.prototype.forEach.call(el.childNodes, function (n) {
        frag.appendChild(n.cloneNode(true));
      });
    } else {
      var level = /^H[1-6]$/.test(el.tagName) ? Number(el.tagName[1]) : 7;
      frag.appendChild(el.cloneNode(true));
      for (var n = nextInSection(el); n; n = nextInSection(n)) {
        if (/^H[1-6]$/.test(n.tagName) && Number(n.tagName[1]) <= level) {
          break;
        }
        frag.appendChild(n.cloneNode(true));
      }
    }
    renameIDs(frag, "--dialog");
    return frag;
  }

  // renameIDs troca os ids de uma cópia para não repetir os do
  // documento, mantendo as referências internas (aria-describedby).
  function renameIDs(frag, suffix) {
    Array.prototype.forEach.call(frag.querySelectorAll("[id]"), function (n) {
      n.id += suffix;
    });
    Array.prototype.forEach.call(frag.querySelectorAll("[aria-describedby]"), function (n) {
      n.setAttribute("aria-describedby", n.getAttribute("aria-describedby") + suffix);
    });
  }

  // nextInSection segue para o próximo elemento; no fim de um slide,
  // continua no seguinte se ele começa num "---" (o corte não fecha a
  // seção, só o slide).
  function nextInSection(n) {
    if (n.nextElementSibling) {
      return n.nextElementSibling;
    }
    var slide = n.parentElement;
    var next = slide && slide.classList.contains("gb-slide") ? slide.nextElementSibling : null;
    return next && next.firstElementChild && next.firstElementChild.tagName === "HR" ? next.firstElementChild : null;
  }

  function openDialog(hash, opener) {
    var el = element(hash);
    if (!el) {
      return;
    }
    dialogBody.replaceChildren(section(el));
    if (!dialog.open) {
      dialogOpener = opener;
      dialog.showModal();
    }
    dialog.scrollTop = 0;
  }

  doc.addEventListener("click", function (e) {
    var a = e.target.closest("a[data-gb]");
    if (!a || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
      return;
    }
    var hash = a.getAttribute("href");
    if (a.dataset.gb === "dialog") {
      e.preventDefault();
      openDialog(hash, a);
      return;
    }
    if (a.dataset.gb === "nav") {
      if (dialog.open) {
        dialog.close();
      }
      if (hash !== location.hash) {
        backStack.push({ hash: location.hash || homeHash(), scrollY: window.scrollY, slide: presenting ? slideIndex : null });
      }
    }
  });

  backButtons.forEach(function (b) {
    b.addEventListener("click", function () {
      var entry = backStack.pop();
      if (!entry) {
        return;
      }
      returning = entry;
      if (location.hash === entry.hash) {
        route();
      } else {
        location.hash = entry.hash;
      }
    });
  });

  dialog.querySelector(".gb-dialog-close").addEventListener("click", function () {
    dialog.close();
  });
  dialog.addEventListener("click", function (e) {
    if (e.target === dialog) {
      dialog.close(); // clique fora do conteúdo
    }
  });
  dialog.addEventListener("close", function () {
    dialogBody.replaceChildren();
    if (dialogOpener) {
      dialogOpener.focus();
      dialogOpener = null;
    }
  });

  // "Comece por aqui": mostra a trilha do perfil escolhido e lembra a
  // escolha quando o navegador deixa.
  var picker = doc.getElementById("gb-profile");
  if (picker) {
    var profiles = Array.prototype.slice.call(doc.querySelectorAll(".gb-profile"));
    var showProfile = function () {
      profiles.forEach(function (s) {
        s.hidden = s.dataset.profile !== picker.value;
      });
    };
    try {
      var saved = localStorage.getItem(data.storageKey + ":profile");
      if (saved && profiles.some(function (s) { return s.dataset.profile === saved; })) {
        picker.value = saved;
      }
    } catch (e) {
      // Sem armazenamento (file:// em alguns navegadores): fica o primeiro.
    }
    picker.closest(".gb-profile-picker").hidden = false;
    showProfile();
    picker.addEventListener("change", function () {
      showProfile();
      try {
        localStorage.setItem(data.storageKey + ":profile", picker.value);
      } catch (e) {
        // idem
      }
    });
  }

  // Busca (docs/spec.md, seção 9.3): o índice MiniSearch é montado aqui,
  // a partir dos documentos gerados no build (ADR 0002).
  var searchBox = doc.querySelector(".gb-search");
  if (searchBox && window.MiniSearch && data.search) {
    var input = doc.getElementById("gb-search-input");
    var results = doc.getElementById("gb-search-results");
    var index = new window.MiniSearch({
      fields: ["title", "section", "text"],
      processTerm: function (term) { return fold(term); },
      searchOptions: { boost: { title: 3, section: 2 }, prefix: true, fuzzy: 0.2, combineWith: "AND" }
    });
    var docsByID = {};
    data.search.forEach(function (d) { docsByID[d.id] = d; });
    index.addAll(data.search);
    searchBox.hidden = false;
    var selected = -1;

    // snippet devolve um trecho do texto em volta do primeiro termo
    // encontrado, com os termos destacados.
    var snippet = function (text, terms) {
      var folded = fold(text);
      var first = text.length;
      terms.forEach(function (t) {
        var i = folded.indexOf(t);
        if (i >= 0 && i < first) { first = i; }
      });
      if (first === text.length) { first = 0; }
      var start = Math.max(0, first - 60);
      var end = Math.min(text.length, start + 160);
      var frag = doc.createDocumentFragment();
      var pos = start;
      if (start > 0) { frag.appendChild(doc.createTextNode("…")); }
      while (pos < end) {
        var next = end, len = 0;
        terms.forEach(function (t) {
          var i = folded.indexOf(t, pos);
          if (i >= 0 && i < next) { next = i; len = t.length; }
        });
        frag.appendChild(doc.createTextNode(text.slice(pos, next)));
        if (len > 0) {
          var mark = doc.createElement("mark");
          mark.textContent = text.slice(next, Math.min(end, next + len));
          frag.appendChild(mark);
        }
        pos = next + len;
      }
      if (end < text.length) { frag.appendChild(doc.createTextNode("…")); }
      return frag;
    };

    var select = function (i) {
      var links = results.querySelectorAll("a");
      selected = links.length ? (i + links.length) % links.length : -1;
      Array.prototype.forEach.call(links, function (a, j) {
        a.classList.toggle("gb-selected", j === selected);
      });
      if (selected >= 0) { links[selected].scrollIntoView({ block: "nearest" }); }
    };

    var hideResults = function () {
      results.hidden = true;
      selected = -1;
    };

    var render = function () {
      var query = input.value.trim();
      results.replaceChildren();
      if (!query) {
        hideResults();
        return;
      }
      var found = index.search(query).slice(0, 20);
      if (found.length === 0) {
        var empty = doc.createElement("li");
        empty.className = "gb-search-empty";
        empty.textContent = input.dataset.noResults;
        results.appendChild(empty);
      }
      found.forEach(function (r) {
        var d = docsByID[r.id];
        var li = doc.createElement("li");
        var a = doc.createElement("a");
        a.href = "#/" + d.id;
        a.dataset.gb = "nav";
        var title = doc.createElement("strong");
        title.textContent = d.section || d.title;
        a.appendChild(title);
        if (d.section) {
          var page = doc.createElement("span");
          page.className = "gb-muted";
          page.textContent = d.title;
          a.appendChild(page);
        }
        if (d.text) {
          var text = doc.createElement("span");
          text.appendChild(snippet(d.text, r.terms.map(fold).filter(function (t) { return t.length > 0; })));
          a.appendChild(text);
        }
        li.appendChild(a);
        results.appendChild(li);
      });
      results.hidden = false;
      select(-1);
    };

    input.addEventListener("input", render);
    input.addEventListener("keydown", function (e) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        select(selected + (e.key === "ArrowDown" ? 1 : -1));
      } else if (e.key === "Enter") {
        var target = results.querySelectorAll("a")[Math.max(selected, 0)];
        if (target) {
          e.preventDefault();
          target.click();
        }
      } else if (e.key === "Escape") {
        input.value = "";
        hideResults();
      }
    });
    results.addEventListener("click", function (e) {
      if (e.target.closest("a")) {
        input.value = "";
        hideResults();
      }
    });
    doc.addEventListener("click", function (e) {
      if (!searchBox.contains(e.target)) { hideResults(); }
    });
    // "/" leva à busca, como em muitos sites de documentação.
    doc.addEventListener("keydown", function (e) {
      var typing = /^(INPUT|TEXTAREA|SELECT)$/.test(doc.activeElement.tagName);
      if (e.key === "/" && !typing && !dialog.open && !isOpen(videoDialog) && !presenting) {
        e.preventDefault();
        input.focus();
      }
    });
  }

  // Modo apresentação (docs/spec.md, seção 9.3). A página continua a
  // mesma (ADR 0001): os slides são cópias dos .gb-slide do artigo
  // atual, mostradas num palco de 1280×720 px lógicos, escalado para
  // caber na tela.
  var SLIDE_WIDTH = 1280;
  var SLIDE_HEIGHT = 720;
  var MIN_FIT = 0.6;
  var presentButton = doc.querySelector(".gb-present");
  var presentation = doc.querySelector(".gb-presentation");
  var inPresentation = function (selector) {
    return presentation ? presentation.querySelector(selector) : null;
  };
  var stage = inPresentation(".gb-stage");
  var frame = inPresentation(".gb-stage-frame");
  var stageContent = inPresentation(".gb-stage-content");
  var counter = inPresentation(".gb-slide-count");
  var prevButton = inPresentation(".gb-prev");
  var nextButton = inPresentation(".gb-next");
  var fullscreenButton = inPresentation(".gb-fullscreen");
  var listToggle = inPresentation(".gb-slide-list-toggle");
  var slideList = doc.getElementById("gb-slide-list");

  // loadSlides lê os slides do artigo; artigo sem cortes (como "Comece
  // por aqui") é um slide só.
  function loadSlides(article) {
    var found = Array.prototype.filter.call(article.children, function (c) {
      return c.classList.contains("gb-slide");
    });
    slides = found.length ? found : [article];
    slideList.replaceChildren();
    slides.forEach(function (slide, i) {
      var heading = slide.querySelector("h1, h2, h3");
      var li = doc.createElement("li");
      var button = doc.createElement("button");
      button.type = "button";
      button.textContent = heading ? heading.textContent : counter.dataset.slideOf.replace("{n}", i + 1).replace("{total}", slides.length);
      if (i === 0 && article.hasAttribute("data-has-video")) {
        var icon = doc.createElement("span");
        icon.className = "gb-has-video";
        icon.title = slideList.dataset.hasVideo;
        var label = doc.createElement("span");
        label.className = "gb-visually-hidden";
        label.textContent = slideList.dataset.hasVideo;
        icon.appendChild(label);
        button.appendChild(icon);
      }
      button.addEventListener("click", function () {
        showSlide(i);
        stage.focus();
      });
      li.appendChild(button);
      slideList.appendChild(li);
    });
  }

  // slideOf devolve o slide que contém el (seção apontada pelo hash).
  function slideOf(el) {
    for (var i = 0; i < slides.length; i++) {
      if (slides[i].contains(el)) {
        return i;
      }
    }
    return 0;
  }

  function showSlide(i) {
    slideIndex = Math.max(0, Math.min(i, slides.length - 1));
    var copy = doc.createDocumentFragment();
    Array.prototype.forEach.call(slides[slideIndex].childNodes, function (n) {
      copy.appendChild(n.cloneNode(true));
    });
    renameIDs(copy, "--slide");
    endTour();
    stageContent.replaceChildren(copy);
    stageContent.classList.remove("gb-enter");
    void stageContent.offsetWidth; // reinicia a animação de entrada
    stageContent.classList.add("gb-enter");
    fit();
    counter.textContent = counter.dataset.slideOf.replace("{n}", slideIndex + 1).replace("{total}", slides.length);
    prevButton.disabled = slideIndex === 0;
    nextButton.disabled = slideIndex === slides.length - 1;
    Array.prototype.forEach.call(slideList.querySelectorAll("button"), function (b, j) {
      if (j === slideIndex) {
        b.setAttribute("aria-current", "true");
      } else {
        b.removeAttribute("aria-current");
      }
    });
    stage.scrollTop = 0;
  }

  // fit escala o slide para a largura do palco e, se o conteúdo passar
  // da altura, reduz até 60%; abaixo disso, o palco rola.
  function fit() {
    if (!presenting) {
      return;
    }
    stageContent.style.transform = "none";
    var base = stage.clientWidth / SLIDE_WIDTH;
    var height = stageContent.offsetHeight;
    var shrink = Math.min(1, SLIDE_HEIGHT / height);
    var scroll = shrink < MIN_FIT;
    var scale = base * (scroll ? MIN_FIT : shrink);
    stageContent.style.transform = "scale(" + scale + ")";
    frame.style.width = SLIDE_WIDTH * scale + "px";
    frame.style.height = height * scale + "px";
    stage.classList.toggle("gb-scroll", scroll);
  }

  function present() {
    var article = doc.querySelector(".gb-page.gb-current");
    if (!article) {
      return;
    }
    presenting = true;
    presentation.hidden = false;
    doc.documentElement.classList.add("gb-presenting");
    loadSlides(article);
    showSlide(slideOf(element(location.hash) || article));
    stage.focus();
  }

  function exitPresentation() {
    if (doc.fullscreenElement) {
      doc.exitFullscreen();
    }
    presenting = false;
    presentation.hidden = true;
    doc.documentElement.classList.remove("gb-presenting");
    stageContent.replaceChildren();
    // A leitura continua no ponto do slide em que a apresentação parou.
    if (slides[slideIndex] && slides[slideIndex].classList.contains("gb-slide")) {
      slides[slideIndex].scrollIntoView();
    }
    presentButton.focus({ preventScroll: true });
  }

  function toggleFullscreen() {
    if (doc.fullscreenElement) {
      doc.exitFullscreen();
    } else if (presentation.requestFullscreen) {
      presentation.requestFullscreen().catch(function () {});
    }
  }

  if (presentation) {
    presentButton.hidden = false;
    presentButton.addEventListener("click", present);
    presentation.querySelector(".gb-exit").addEventListener("click", exitPresentation);
    prevButton.addEventListener("click", function () { showSlide(slideIndex - 1); });
    nextButton.addEventListener("click", function () { showSlide(slideIndex + 1); });
    if (doc.fullscreenEnabled) {
      fullscreenButton.addEventListener("click", toggleFullscreen);
    } else {
      fullscreenButton.hidden = true;
    }
    listToggle.addEventListener("click", function () {
      var open = listToggle.getAttribute("aria-expanded") !== "true";
      listToggle.setAttribute("aria-expanded", String(open));
      slideList.hidden = !open;
      fit();
    });
    window.addEventListener("resize", fit);
    doc.addEventListener("fullscreenchange", fit);

    doc.addEventListener("keydown", function (e) {
      if (!presenting || dialog.open || isOpen(videoDialog) || e.ctrlKey || e.metaKey || e.altKey) {
        return;
      }
      // Espaço e Enter em botão ou link ficam com o próprio elemento.
      var onControl = e.target.closest && e.target.closest("button, a, input, select, textarea");
      switch (e.key) {
        case "ArrowRight":
        case "PageDown":
          showSlide(slideIndex + 1);
          break;
        case "ArrowLeft":
        case "PageUp":
          showSlide(slideIndex - 1);
          break;
        case " ":
          if (onControl) {
            return;
          }
          showSlide(slideIndex + (e.shiftKey ? -1 : 1));
          break;
        case "Home":
          showSlide(0);
          break;
        case "End":
          showSlide(slides.length - 1);
          break;
        case "f":
        case "F":
          if (onControl && /^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName)) {
            return;
          }
          toggleFullscreen();
          break;
        case "Escape":
          // Em tela cheia, o navegador já usa o Esc para sair dela.
          if (!doc.fullscreenElement) {
            exitPresentation();
          }
          break;
        default:
          return;
      }
      e.preventDefault();
    });
  }

  // Capturas com marcadores (docs/spec.md, seção 6.3). Os marcadores
  // são posicionados aqui, uma vez, antes de qualquer cópia para slide
  // ou diálogo; os eventos são delegados e valem também nas cópias.
  Array.prototype.forEach.call(doc.querySelectorAll(".gb-shot"), function (fig) {
    Array.prototype.forEach.call(fig.querySelectorAll(".gb-marker"), function (m) {
      m.style.left = Number(m.dataset.x) * 100 + "%";
      m.style.top = Number(m.dataset.y) * 100 + "%";
      m.hidden = false;
    });
    fig.querySelector(".gb-tour").hidden = false;
  });

  // steps devolve os itens da lista numerada que vem logo depois da
  // captura: o marcador n corresponde ao item n.
  function steps(fig) {
    var list = fig.nextElementSibling;
    return list && list.hasAttribute("data-gb-steps") ? list.children : [];
  }

  function highlight(fig, n) {
    Array.prototype.forEach.call(fig.querySelectorAll(".gb-marker"), function (m) {
      m.classList.toggle("gb-active", Number(m.dataset.n) === n);
    });
    [steps(fig), fig.querySelector(".gb-shot-points").children].forEach(function (items) {
      Array.prototype.forEach.call(items, function (li, i) {
        li.classList.toggle("gb-highlight", i + 1 === n);
      });
    });
  }

  // Visita guiada: amplia a captura em 2× com o marcador atual no
  // centro, sem sair da borda da imagem.
  var tour = null;

  function tourStep(i) {
    var markers = tour.fig.querySelectorAll(".gb-marker");
    tour.i = Math.max(0, Math.min(i, markers.length - 1));
    var m = markers[tour.i];
    var clamp = function (v) { return Math.max(-1, Math.min(0, v)); };
    var dx = clamp(0.5 - 2 * Number(m.dataset.x));
    var dy = clamp(0.5 - 2 * Number(m.dataset.y));
    tour.fig.querySelector(".gb-shot-zoom").style.transform = "translate(" + dx * 100 + "%, " + dy * 100 + "%) scale(2)";
    tour.fig.querySelector(".gb-tour-text").textContent = m.getAttribute("aria-label");
    tour.fig.querySelector(".gb-tour-prev").disabled = tour.i === 0;
    tour.fig.querySelector(".gb-tour-next").disabled = tour.i === markers.length - 1;
    highlight(tour.fig, tour.i + 1);
  }

  function startTour(fig, opener) {
    endTour();
    tour = { fig: fig, i: 0, opener: opener };
    fig.classList.add("gb-touring");
    fig.querySelector(".gb-tour-panel").hidden = false;
    tourStep(0);
    fig.querySelector(".gb-tour-next").focus();
  }

  function endTour() {
    if (!tour) {
      return;
    }
    var t = tour;
    tour = null;
    t.fig.classList.remove("gb-touring");
    t.fig.querySelector(".gb-shot-zoom").style.transform = "";
    t.fig.querySelector(".gb-tour-panel").hidden = true;
    if (t.opener && doc.contains(t.opener)) {
      t.opener.focus();
    }
  }

  doc.addEventListener("click", function (e) {
    var target = e.target;
    var fig = target.closest(".gb-shot");
    if (target.closest(".gb-marker")) {
      highlight(fig, Number(target.closest(".gb-marker").dataset.n));
    } else if (target.closest(".gb-tour")) {
      startTour(fig, target.closest(".gb-tour"));
    } else if (target.closest(".gb-tour-next")) {
      tourStep(tour.i + 1);
    } else if (target.closest(".gb-tour-prev")) {
      tourStep(tour.i - 1);
    } else if (target.closest(".gb-tour-close")) {
      endTour();
    } else if (target.closest(".gb-shot-points > li")) {
      var li = target.closest("li");
      highlight(fig, Array.prototype.indexOf.call(li.parentNode.children, li) + 1);
    } else {
      var step = target.closest("ol[data-gb-steps] > li");
      var shotFig = step && step.parentNode.previousElementSibling;
      if (shotFig && shotFig.classList.contains("gb-shot")) {
        highlight(shotFig, Array.prototype.indexOf.call(step.parentNode.children, step) + 1);
      }
    }
  });

  // Durante a visita, setas e Esc são dela, antes do modo apresentação.
  doc.addEventListener("keydown", function (e) {
    if (!tour || dialog.open || isOpen(videoDialog)) {
      return;
    }
    if (e.key === "ArrowRight" || e.key === "ArrowDown") {
      tourStep(tour.i + 1);
    } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
      tourStep(tour.i - 1);
    } else if (e.key === "Escape") {
      endTour();
    } else {
      return;
    }
    e.preventDefault();
    e.stopImmediatePropagation();
  }, true);
  window.addEventListener("hashchange", endTour);

  // Vídeo por tarefa (docs/spec.md, seção 7.2): diálogo grande com os
  // controles nativos, sem sincronia com o texto. Sem JavaScript, o link
  // abre o arquivo no navegador.
  if (videoDialog) {
    var videoBody = videoDialog.querySelector(".gb-video-body");
    var videoOpener = null;

    doc.addEventListener("click", function (e) {
      var link = e.target.closest("a.gb-video-link");
      if (!link || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
        return;
      }
      e.preventDefault();
      var video = doc.createElement("video");
      video.controls = true;
      video.preload = "none";
      video.src = link.getAttribute("href");
      if (link.dataset.poster) {
        video.poster = link.dataset.poster;
      }
      video.setAttribute("aria-label", link.textContent);
      videoBody.replaceChildren(video);
      videoOpener = link;
      videoDialog.showModal();
      video.focus();
    });
    videoDialog.querySelector(".gb-video-close").addEventListener("click", function () {
      videoDialog.close();
    });
    videoDialog.addEventListener("click", function (e) {
      if (e.target === videoDialog) {
        videoDialog.close();
      }
    });
    videoDialog.addEventListener("close", function () {
      // Fechar pausa o vídeo e para o download.
      var video = videoBody.querySelector("video");
      if (video) {
        video.pause();
        video.removeAttribute("src");
        video.load();
      }
      videoBody.replaceChildren();
      if (videoOpener && doc.contains(videoOpener)) {
        videoOpener.focus();
      }
      videoOpener = null;
    });
  }

  // Diagramas Mermaid (docs/spec.md, seção 8). O guidebook controla a
  // inicialização: nada roda sozinho, e cada diagrama é desenhado quando
  // a página dele aparece (desenhar escondido erra o tamanho). Diagrama
  // que falha mostra o texto original e a mensagem do Mermaid.
  if (window.mermaid) {
    window.mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "default"
    });
  }

  function renderDiagrams(article) {
    if (!window.mermaid) {
      return;
    }
    Array.prototype.forEach.call(article.querySelectorAll("pre.mermaid:not([data-gb-source])"), function (el) {
      el.dataset.gbSource = el.textContent;
      window.mermaid.run({ nodes: [el] }).catch(function (err) {
        el.textContent = el.dataset.gbSource;
        el.classList.add("gb-mermaid-failed");
        var message = doc.createElement("p");
        message.className = "gb-mermaid-error";
        message.textContent = (err && err.message) || String(err);
        el.after(message);
      });
    });
  }

  // Regras de negócio: filtro por texto e por módulo. Cada regra é um
  // slide do módulo (cada "## " abre um).
  var rulesFilter = doc.querySelector(".gb-rules-filter");
  if (rulesFilter) {
    var rulesText = doc.getElementById("gb-rules-text");
    var rulesModule = doc.getElementById("gb-rules-module");
    var rulesNone = doc.querySelector(".gb-rules-none");
    var filterRules = function () {
      var terms = fold(rulesText.value).split(/\s+/).filter(Boolean);
      var shown = 0;
      Array.prototype.forEach.call(doc.querySelectorAll(".gb-rules-module"), function (mod) {
        var moduleOK = !rulesModule.value || mod.dataset.module === rulesModule.value;
        var visible = 0;
        Array.prototype.forEach.call(mod.querySelectorAll(":scope > .gb-slide"), function (slide) {
          if (!slide.querySelector(":scope > h2")) {
            return; // título e introdução do módulo
          }
          var text = fold(slide.textContent);
          var ok = moduleOK && terms.every(function (t) { return text.indexOf(t) >= 0; });
          slide.hidden = !ok;
          visible += ok ? 1 : 0;
        });
        mod.hidden = visible === 0;
        shown += visible;
      });
      rulesNone.hidden = shown > 0;
    };
    rulesFilter.hidden = false;
    rulesText.addEventListener("input", filterRules);
    rulesModule.addEventListener("change", filterRules);
  }

  window.addEventListener("hashchange", route);
  route();
})();
