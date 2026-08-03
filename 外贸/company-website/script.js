const header = document.querySelector("[data-header]");
const menuButton = document.querySelector("[data-menu-button]");
const mobileNav = document.querySelector("[data-mobile-nav]");
const pageProgress = document.querySelector("[data-page-progress]");
const hero = document.querySelector("[data-hero]");
const themeSections = Array.from(document.querySelectorAll("[data-nav-theme]"));
const navLinks = Array.from(document.querySelectorAll(".desktop-nav a, .mobile-nav a"));
const companyName = "常无（四川）信息科技有限责任公司";
const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

const clamp = (value, min = 0, max = 1) => Math.min(max, Math.max(min, value));

const closeMenu = () => {
  if (!menuButton || !mobileNav) return;
  menuButton.setAttribute("aria-expanded", "false");
  menuButton.setAttribute("aria-label", "打开导航");
  mobileNav.classList.remove("open");
  header?.classList.remove("menu-visible");
  document.body.classList.remove("menu-open");
};

menuButton?.addEventListener("click", () => {
  const willOpen = menuButton.getAttribute("aria-expanded") !== "true";
  menuButton.setAttribute("aria-expanded", String(willOpen));
  menuButton.setAttribute("aria-label", willOpen ? "关闭导航" : "打开导航");
  mobileNav?.classList.toggle("open", willOpen);
  header?.classList.toggle("menu-visible", willOpen);
  document.body.classList.toggle("menu-open", willOpen);
});

mobileNav?.querySelectorAll("a").forEach((link) => link.addEventListener("click", closeMenu));

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") closeMenu();
});

window.addEventListener("resize", () => {
  if (window.innerWidth > 720) closeMenu();
});

const businessStory = document.querySelector("[data-business-story]");
const businessSteps = Array.from(document.querySelectorAll("[data-business-step]"));
const businessJumps = Array.from(document.querySelectorAll("[data-business-jump]"));
const businessCaption = document.querySelector("[data-business-caption]");
const businessNumber = document.querySelector("[data-business-number]");

const fulfillmentStory = document.querySelector("[data-fulfillment-story]");
const fulfillmentSteps = Array.from(document.querySelectorAll("[data-fulfillment-step]"));
const fulfillmentJumps = Array.from(document.querySelectorAll("[data-fulfillment-jump]"));
const fulfillmentState = document.querySelector("[data-fulfillment-state]");
const fulfillmentStatusLabel = fulfillmentState?.previousElementSibling;
const fulfillmentStateLabels = ["订单已受理", "货源与成本已核", "行程全段可溯", "妥投且账实相合"];

const activateStep = (steps, controls, index) => {
  steps.forEach((step, stepIndex) => step.classList.toggle("is-active", stepIndex === index));
  controls.forEach((control, controlIndex) => {
    const active = controlIndex === index;
    control.classList.toggle("is-active", active);
    if (control.hasAttribute("aria-pressed")) control.setAttribute("aria-pressed", String(active));
  });
};

const getSceneProgress = (section) => {
  if (!section) return 0;
  const rect = section.getBoundingClientRect();
  const travel = Math.max(1, section.offsetHeight - window.innerHeight);
  return clamp(-rect.top / travel);
};

const updateScrollScene = (section, steps, controls, cssProperty, onChange) => {
  if (!section || steps.length === 0) return;
  const progress = getSceneProgress(section);
  section.style.setProperty(cssProperty, String(progress));
  const activeIndex = Math.min(steps.length - 1, Math.floor(progress * steps.length));

  if (section.dataset.activeIndex !== String(activeIndex)) {
    section.dataset.activeIndex = String(activeIndex);
    activateStep(steps, controls, activeIndex);
    onChange?.(activeIndex);
  }
};

const jumpToSceneStep = (section, index, count) => {
  if (!section) return;
  const sectionTop = window.scrollY + section.getBoundingClientRect().top;
  const travel = Math.max(0, section.offsetHeight - window.innerHeight);
  const progress = (index + 0.5) / count;
  window.scrollTo({ top: sectionTop + travel * progress, behavior: reducedMotion ? "auto" : "smooth" });
};

businessJumps.forEach((button, index) => {
  button.setAttribute("aria-pressed", String(index === 0));
  button.addEventListener("click", () => jumpToSceneStep(businessStory, index, businessSteps.length));
});

fulfillmentJumps.forEach((button, index) => {
  button.setAttribute("aria-label", `查看履约步骤${index + 1}`);
  button.setAttribute("aria-pressed", String(index === 0));
  button.addEventListener("click", () => jumpToSceneStep(fulfillmentStory, index, fulfillmentSteps.length));
});

let frameRequested = false;

const renderScrollState = () => {
  frameRequested = false;
  const maxScroll = Math.max(1, document.documentElement.scrollHeight - window.innerHeight);
  const documentProgress = clamp(window.scrollY / maxScroll);
  pageProgress?.style.setProperty("transform", `scaleX(${documentProgress})`);
  header?.classList.toggle("scrolled", window.scrollY > 24);

  const headerProbe = (header?.offsetHeight || 58) / 2 + 2;
  const currentTheme = themeSections.find((section) => {
    const rect = section.getBoundingClientRect();
    return rect.top <= headerProbe && rect.bottom > headerProbe;
  });
  header?.classList.toggle("on-dark", currentTheme?.dataset.navTheme === "dark");

  let activeTarget = "top";
  navLinks.forEach((link) => {
    const targetId = link.getAttribute("href")?.slice(1);
    const target = targetId ? document.getElementById(targetId) : null;
    if (target && target.getBoundingClientRect().top <= window.innerHeight * 0.42) activeTarget = targetId;
  });
  navLinks.forEach((link) => {
    const active = link.getAttribute("href") === `#${activeTarget}`;
    if (active) link.setAttribute("aria-current", "location");
    else link.removeAttribute("aria-current");
  });

  if (hero && !reducedMotion) {
    const heroProgress = clamp(window.scrollY / Math.max(1, hero.offsetHeight));
    hero.style.setProperty("--hero-progress", String(heroProgress));
  }

  updateScrollScene(
    businessStory,
    businessSteps,
    businessJumps,
    "--business-progress",
    (index) => {
      if (businessCaption) businessCaption.textContent = businessSteps[index]?.dataset.caption || "";
      if (businessNumber) {
        businessNumber.textContent = String(index + 1).padStart(2, "0");
        if (!reducedMotion && businessNumber.animate) {
          businessNumber.animate(
            [
              { opacity: 0, transform: "translateY(24px)" },
              { opacity: 1, transform: "translateY(0)" }
            ],
            { duration: 480, easing: "cubic-bezier(.2,.7,.2,1)" }
          );
        }
      }
    }
  );

  updateScrollScene(
    fulfillmentStory,
    fulfillmentSteps,
    fulfillmentJumps,
    "--fulfillment-progress",
    (index) => {
      if (fulfillmentState) fulfillmentState.textContent = fulfillmentStateLabels[index] || "";
      if (fulfillmentStatusLabel) fulfillmentStatusLabel.textContent = `ORDER STATE / ${String(index + 1).padStart(2, "0")}`;
      if (!reducedMotion && fulfillmentState?.animate) {
        fulfillmentState.animate(
          [
            { opacity: 0, transform: "translateY(8px)" },
            { opacity: 1, transform: "translateY(0)" }
          ],
          { duration: 360, easing: "ease-out" }
        );
      }
    }
  );
};

const requestScrollRender = () => {
  if (frameRequested) return;
  frameRequested = true;
  window.requestAnimationFrame(renderScrollState);
};

window.addEventListener("scroll", requestScrollRender, { passive: true });
window.addEventListener("resize", requestScrollRender);
requestScrollRender();

const productTabs = Array.from(document.querySelectorAll("[data-product-tab]"));
const productPanels = Array.from(document.querySelectorAll("[data-product-panel]"));
const productVisual = document.querySelector("[data-product-visual]");
const productFocusLabel = document.querySelector("[data-product-focus-label]");
const productLabels = {
  desk: "桌面办公",
  home: "家居收纳",
  travel: "旅行整理",
  beauty: "美妆工具"
};

const selectProduct = (key, moveFocus = false) => {
  productTabs.forEach((tab) => {
    const selected = tab.dataset.productTab === key;
    tab.setAttribute("aria-selected", String(selected));
    tab.tabIndex = selected ? 0 : -1;
    if (selected && moveFocus) tab.focus();
  });

  productPanels.forEach((panel) => {
    const selected = panel.dataset.productPanel === key;
    panel.hidden = !selected;
    if (selected && !reducedMotion && panel.animate) {
      panel.animate(
        [
          { opacity: 0, transform: "translateY(12px)" },
          { opacity: 1, transform: "translateY(0)" }
        ],
        { duration: 420, easing: "cubic-bezier(.2,.7,.2,1)" }
      );
    }
  });

  if (productVisual) productVisual.dataset.focus = key;
  if (productFocusLabel) productFocusLabel.textContent = productLabels[key] || "";
};

productTabs.forEach((tab, index) => {
  tab.addEventListener("click", () => selectProduct(tab.dataset.productTab));
  tab.addEventListener("keydown", (event) => {
    const keys = ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End"];
    if (!keys.includes(event.key)) return;
    event.preventDefault();

    let nextIndex = index;
    if (["ArrowRight", "ArrowDown"].includes(event.key)) nextIndex = (index + 1) % productTabs.length;
    if (["ArrowLeft", "ArrowUp"].includes(event.key)) nextIndex = (index - 1 + productTabs.length) % productTabs.length;
    if (event.key === "Home") nextIndex = 0;
    if (event.key === "End") nextIndex = productTabs.length - 1;
    selectProduct(productTabs[nextIndex].dataset.productTab, true);
  });
});

const revealItems = document.querySelectorAll(".reveal");

if (reducedMotion || !("IntersectionObserver" in window)) {
  revealItems.forEach((item) => item.classList.add("visible"));
} else {
  const revealObserver = new IntersectionObserver((entries, observer) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) return;
      entry.target.classList.add("visible");
      observer.unobserve(entry.target);
    });
  }, { threshold: 0.13, rootMargin: "0px 0px -48px" });

  revealItems.forEach((item) => revealObserver.observe(item));
}

const opsConsole = document.querySelector(".ops-console");
const opsRows = Array.from(document.querySelectorAll(".ops-row"));
let opsRowIndex = 0;
let opsTimer;

const stopOpsCycle = () => {
  if (!opsTimer) return;
  window.clearInterval(opsTimer);
  opsTimer = undefined;
};

const startOpsCycle = () => {
  if (reducedMotion || opsTimer || opsRows.length < 2 || document.hidden) return;
  opsTimer = window.setInterval(() => {
    opsRowIndex = (opsRowIndex + 1) % opsRows.length;
    opsRows.forEach((row, index) => row.classList.toggle("is-live", index === opsRowIndex));
  }, 1800);
};

if (opsConsole && "IntersectionObserver" in window) {
  const opsObserver = new IntersectionObserver((entries) => {
    if (entries[0]?.isIntersecting) startOpsCycle();
    else stopOpsCycle();
  }, { threshold: 0.35 });
  opsObserver.observe(opsConsole);
}

document.addEventListener("visibilitychange", () => {
  if (document.hidden) stopOpsCycle();
  else if (opsConsole?.getBoundingClientRect().top < window.innerHeight && opsConsole?.getBoundingClientRect().bottom > 0) startOpsCycle();
});

const copyText = async (text) => {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const field = document.createElement("textarea");
  field.value = text;
  field.setAttribute("readonly", "");
  field.style.position = "fixed";
  field.style.opacity = "0";
  document.body.appendChild(field);
  field.select();
  const copied = document.execCommand("copy");
  field.remove();
  if (!copied) throw new Error("Copy command failed");
};

document.querySelector("[data-copy-company]")?.addEventListener("click", async (event) => {
  const button = event.currentTarget;
  const originalText = button.textContent;
  try {
    await copyText(companyName);
    button.textContent = "已复制公司全称";
  } catch {
    button.textContent = "复制失败，请手动选择";
  }

  window.setTimeout(() => {
    button.textContent = originalText;
  }, 1800);
});

document.querySelectorAll("[data-year]").forEach((element) => {
  element.textContent = String(new Date().getFullYear());
});
