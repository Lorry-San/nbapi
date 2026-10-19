/*
 * Liquid Glass for small floating controls: the header island and the language toggle.
 *
 * The look follows liquidGL (https://github.com/naughtyduk/liquidGL, MIT): bevel refraction,
 * light frost, chromatic dispersion, a specular rim and a tint. liquidGL refracts a DOM
 * snapshot and does not redraw canvases, so glass over the animated WebGL background would
 * show a frozen frame. This file refracts the live backdrop through an SVG displacement
 * filter instead. Live refraction needs Chromium (Chrome, Edge); other engines get frosted
 * glass with the same rim.
 */
(() => {
  const SVG_NS = "http://www.w3.org/2000/svg";
  const brands = (navigator.userAgentData && navigator.userAgentData.brands) || [];
  const LIVE = brands.some((b) => /Chromium/.test(b.brand)) && CSS.supports("backdrop-filter", "url(#lg)");
  const REDUCE = matchMedia("(prefers-reduced-motion: reduce)").matches;

  const CSS_TEXT = `
.lg-glass{-webkit-backdrop-filter:var(--lg-filter,blur(14px) saturate(170%));backdrop-filter:var(--lg-filter,blur(14px) saturate(170%))}
.lg-glass.lg-off{-webkit-backdrop-filter:none;backdrop-filter:none}
.lg-rim{position:absolute;inset:0;border-radius:inherit;pointer-events:none;box-shadow:inset 0 1px 1px rgba(255,255,255,.3),inset 0 -1px 1px rgba(255,255,255,.1),inset 0 0 16px rgba(255,255,255,.07),0 8px 24px -8px rgba(0,0,0,.5)}
.lg-rim::before{content:"";position:absolute;inset:0;border-radius:inherit;padding:1px;background:linear-gradient(var(--lg-angle,315deg),rgba(255,255,255,.3),rgba(255,255,255,.05) 32%,rgba(255,255,255,0) 50%,rgba(255,255,255,.06) 68%,rgba(255,255,255,.78));-webkit-mask:linear-gradient(#000 0 0) content-box,linear-gradient(#000 0 0);-webkit-mask-composite:xor;mask:linear-gradient(#000 0 0) content-box exclude,linear-gradient(#000 0 0)}
.lg-rim::after{content:"";position:absolute;inset:0;border-radius:inherit;background:linear-gradient(180deg,rgba(255,255,255,.12),rgba(255,255,255,0) 48%)}
.lg-thumb{position:absolute;top:3px;bottom:3px;left:0;z-index:0;width:var(--w,0);translate:var(--x,0) 0;border-radius:999px;background:rgba(255,255,255,.13);pointer-events:none;transition:translate .55s cubic-bezier(.3,1.35,.45,1),width .55s cubic-bezier(.3,1.35,.45,1)}
.lg-thumb.lg-wobble{animation:lg-wobble .55s cubic-bezier(.3,1.2,.4,1)}
@keyframes lg-wobble{0%{scale:1}35%{scale:1.22 .82}70%{scale:.96 1.05}100%{scale:1}}
@media (prefers-reduced-motion:reduce){.lg-thumb{transition:none}.lg-thumb.lg-wobble{animation:none}}`;

  let defs = null;
  let seq = 0;

  function setup() {
    if (defs) return defs;
    const style = document.createElement("style");
    style.textContent = CSS_TEXT;
    document.head.appendChild(style);
    const svg = document.createElementNS(SVG_NS, "svg");
    svg.setAttribute("aria-hidden", "true");
    svg.style.cssText = "position:absolute;width:0;height:0;overflow:hidden;pointer-events:none";
    defs = document.createElementNS(SVG_NS, "defs");
    svg.appendChild(defs);
    document.body.appendChild(svg);
    return defs;
  }

  function svgNode(tag, attrs) {
    const node = document.createElementNS(SVG_NS, tag);
    for (const name in attrs) node.setAttribute(name, attrs[name]);
    return node;
  }

  // Displacement map for a rounded rectangle: samples are pulled inward across the bevel,
  // steeply at the rim, and optionally scaled toward the centre to magnify.
  function displacementMap(w, h, radius, o) {
    const cw = Math.max(2, Math.ceil(w * o.mapScale));
    const ch = Math.max(2, Math.ceil(h * o.mapScale));
    const hw = w / 2, hh = h / 2;
    const r = Math.min(radius, hw, hh);
    const bevel = Math.max(1, Math.min(o.bevel, hw, hh));
    const shrink = 1 / o.magnify - 1;
    const offsets = new Float32Array(cw * ch * 2);
    let max = 0.5;
    for (let y = 0, i = 0; y < ch; y++) {
      const py = ((y + 0.5) / ch) * h - hh;
      for (let x = 0; x < cw; x++, i += 2) {
        const px = ((x + 0.5) / cw) * w - hw;
        const qx = Math.abs(px) - (hw - r);
        const qy = Math.abs(py) - (hh - r);
        const outside = Math.hypot(Math.max(qx, 0), Math.max(qy, 0));
        const inward = r - outside - Math.min(Math.max(qx, qy), 0);
        let nx = 0, ny = 0;
        if (qx > 0 && qy > 0) {
          const len = Math.hypot(qx, qy) || 1;
          nx = (qx / len) * Math.sign(px);
          ny = (qy / len) * Math.sign(py);
        } else if (qx > qy) {
          nx = Math.sign(px);
        } else {
          ny = Math.sign(py);
        }
        const t = Math.min(1, Math.max(0, inward / bevel));
        const pull = -o.depth * (1 - t) ** 3;
        const dx = nx * pull + px * shrink;
        const dy = ny * pull + py * shrink;
        offsets[i] = dx;
        offsets[i + 1] = dy;
        max = Math.max(max, Math.abs(dx), Math.abs(dy));
      }
    }
    const canvas = document.createElement("canvas");
    canvas.width = cw;
    canvas.height = ch;
    const ctx = canvas.getContext("2d");
    const image = ctx.createImageData(cw, ch);
    const data = image.data;
    for (let p = 0, i = 0; p < data.length; p += 4, i += 2) {
      data[p] = 128 + Math.round((offsets[i] / max) * 127);
      data[p + 1] = 128 + Math.round((offsets[i + 1] / max) * 127);
      data[p + 2] = 128;
      data[p + 3] = 255;
    }
    ctx.putImageData(image, 0, 0);
    return { url: canvas.toDataURL("image/png"), max };
  }

  const CHANNELS = [
    "1 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1 0",
    "0 0 0 0 0  0 1 0 0 0  0 0 0 0 0  0 0 0 1 0",
    "0 0 0 0 0  0 0 0 0 0  0 0 1 0 0  0 0 0 1 0",
  ];

  function buildFilter(id, o) {
    const filter = svgNode("filter", {
      id, x: 0, y: 0, width: 1, height: 1,
      filterUnits: "userSpaceOnUse", "color-interpolation-filters": "sRGB",
    });
    const image = svgNode("feImage", { x: 0, y: 0, width: 1, height: 1, preserveAspectRatio: "none", result: "map" });
    filter.append(svgNode("feGaussianBlur", { in: "SourceGraphic", stdDeviation: o.frost, result: "frost" }), image);
    const maps = [];
    if (o.aberration > 0) {
      // Blue bends further than red, so dispersion appears where the bevel refracts most.
      CHANNELS.forEach((matrix, i) => {
        const map = svgNode("feDisplacementMap", { in: "frost", in2: "map", xChannelSelector: "R", yChannelSelector: "G", result: `d${i}` });
        filter.append(map, svgNode("feColorMatrix", { in: `d${i}`, type: "matrix", values: matrix, result: `c${i}` }));
        maps.push([map, 1 + i * o.aberration]);
      });
      filter.append(
        svgNode("feComposite", { in: "c0", in2: "c1", operator: "arithmetic", k2: 1, k3: 1, result: "c01" }),
        svgNode("feComposite", { in: "c01", in2: "c2", operator: "arithmetic", k2: 1, k3: 1, result: "glass" }),
      );
    } else {
      const map = svgNode("feDisplacementMap", { in: "frost", in2: "map", xChannelSelector: "R", yChannelSelector: "G", result: "glass" });
      filter.append(map);
      maps.push([map, 1]);
    }
    filter.append(svgNode("feColorMatrix", { in: "glass", type: "saturate", values: o.saturate }));
    return { filter, image, maps };
  }

  /**
   * Turn a positioned element into a glass pane. The element keeps its own background as tint.
   * Options: bevel and depth (px), magnify, frost (px), saturate, aberration (0-1).
   */
  function attach(node, options) {
    const o = { bevel: 16, depth: 12, magnify: 1, frost: 1.2, saturate: 1.6, aberration: 0.05, mapScale: 0.5, ...options };
    setup();
    node.classList.add("lg-glass");
    const rim = document.createElement("span");
    rim.className = "lg-rim";
    rim.setAttribute("aria-hidden", "true");
    node.appendChild(rim);

    let pointer = null, lightFrame = 0;
    const onPointer = (e) => {
      pointer = e;
      if (lightFrame) return;
      lightFrame = requestAnimationFrame(() => {
        lightFrame = 0;
        const r = node.getBoundingClientRect();
        if (!r.width) return;
        const angle = (Math.atan2(pointer.clientY - (r.top + r.height / 2), pointer.clientX - (r.left + r.width / 2)) * 180) / Math.PI + 90;
        node.style.setProperty("--lg-angle", `${angle.toFixed(1)}deg`);
      });
    };
    if (!REDUCE) addEventListener("pointermove", onPointer, { passive: true });

    if (!LIVE) {
      node.classList.add("lg-fallback");
      return { live: false, destroy() { removeEventListener("pointermove", onPointer); rim.remove(); } };
    }

    const id = `lg-${++seq}`;
    const { filter, image, maps } = buildFilter(id, o);
    defs.appendChild(filter);
    // A custom property rather than an inline backdrop-filter, so page CSS (.lg-off) can switch it off.
    node.style.setProperty("--lg-filter", `url(#${id})`);

    let w = 0, h = 0, settle = 0, destroyed = false;
    const regenerate = () => {
      if (destroyed || !w || !h) return;
      const radius = parseFloat(getComputedStyle(node).borderTopLeftRadius) || 0;
      const { url, max } = displacementMap(w, h, radius, o);
      const preload = new Image();
      preload.src = url;
      // Swap only after decoding so the filter never runs without a map.
      (preload.decode ? preload.decode() : Promise.resolve()).catch(() => {}).then(() => {
        if (destroyed) return;
        image.setAttribute("href", url);
        for (const [map, k] of maps) map.setAttribute("scale", (2 * max * k).toFixed(2));
      });
    };
    // While the element animates its size the current map is stretched; it is rebuilt once the size settles.
    const observer = new ResizeObserver(() => {
      const nw = node.offsetWidth, nh = node.offsetHeight;
      if (!nw || !nh || (nw === w && nh === h)) return;
      const first = !w;
      w = nw;
      h = nh;
      for (const n of [filter, image]) {
        n.setAttribute("width", w);
        n.setAttribute("height", h);
      }
      clearTimeout(settle);
      settle = setTimeout(regenerate, first ? 0 : 140);
    });
    observer.observe(node);

    return {
      live: true,
      destroy() {
        destroyed = true;
        observer.disconnect();
        clearTimeout(settle);
        removeEventListener("pointermove", onPointer);
        filter.remove();
        rim.remove();
        node.style.removeProperty("--lg-filter");
        node.classList.remove("lg-glass", "lg-fallback");
      },
    };
  }

  /**
   * A segmented control whose selection is a sliding glass thumb.
   * The group needs position:relative; option buttons carry [data-value].
   * Returns { select(value) } and reports user choices through onChange(value).
   */
  function segmented(group, { onChange, glass } = {}) {
    setup();
    const buttons = Array.from(group.querySelectorAll("[data-value]"));
    const thumb = document.createElement("span");
    thumb.className = "lg-thumb";
    thumb.setAttribute("aria-hidden", "true");
    group.prepend(thumb);
    attach(thumb, { bevel: 9, depth: 7, magnify: 1.12, frost: 0.6, aberration: 0.08, ...glass });
    for (const button of buttons) {
      button.style.position = "relative";
      button.style.zIndex = "1";
    }

    let current = null;
    const place = (button, animate) => {
      if (!animate) thumb.style.transition = "none";
      thumb.style.setProperty("--x", `${button.offsetLeft}px`);
      thumb.style.setProperty("--w", `${button.offsetWidth}px`);
      if (!animate) {
        thumb.getBoundingClientRect();
        thumb.style.transition = "";
      }
    };
    const select = (value, animate = true) => {
      const button = buttons.find((b) => b.dataset.value === value);
      if (!button) return;
      const moved = current !== null && current !== button;
      current = button;
      for (const b of buttons) {
        const on = b === button;
        b.classList.toggle("is-active", on);
        b.setAttribute("aria-pressed", String(on));
      }
      place(button, animate && moved);
      if (moved && animate && !REDUCE) {
        thumb.classList.remove("lg-wobble");
        void thumb.offsetWidth;
        thumb.classList.add("lg-wobble");
      }
    };
    thumb.addEventListener("animationend", () => thumb.classList.remove("lg-wobble"));
    for (const button of buttons) {
      button.addEventListener("click", () => {
        if (button === current) return;
        select(button.dataset.value);
        onChange?.(button.dataset.value);
      });
    }
    // Label widths change with the language and the web font, so re-measure without animating.
    new ResizeObserver(() => current && place(current, false)).observe(group);
    return { select };
  }

  window.LiquidGlass = { attach, segmented, live: LIVE };
})();
