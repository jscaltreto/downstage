const navLinks = Array.from(
  document.querySelectorAll(".js-section-nav a[href^='#']"),
);

const sections = navLinks
  .map((link) => {
    const id = link.getAttribute("href");
    return id ? document.querySelector(id) : null;
  })
  .filter(Boolean);

const setActive = () => {
  const offset = window.scrollY + window.innerHeight * 0.24;
  let currentId = sections[0]?.id ?? "";

  for (const section of sections) {
    if (section.offsetTop <= offset) {
      currentId = section.id;
    }
  }

  for (const link of navLinks) {
    const active = link.getAttribute("href") === `#${currentId}`;
    link.classList.toggle("active", active);
  }
};

if (navLinks.length > 0) {
  setActive();
  window.addEventListener("scroll", setActive, { passive: true });
}

const mobileMenuButtons = Array.from(document.querySelectorAll("[data-mobile-menu-toggle]"));

const closeMobileMenu = (button, panel, backdrop) => {
  button.setAttribute("aria-expanded", "false");
  panel.classList.remove("is-open");
  backdrop?.classList.add("hidden");
  window.setTimeout(() => {
    if (button.getAttribute("aria-expanded") === "false") {
      panel.classList.add("hidden");
    }
  }, 200);
};

const openMobileMenu = (button, panel, backdrop) => {
  button.setAttribute("aria-expanded", "true");
  panel.classList.remove("hidden");
  backdrop?.classList.remove("hidden");
  requestAnimationFrame(() => {
    panel.classList.add("is-open");
  });
};

for (const button of mobileMenuButtons) {
  const panelId = button.getAttribute("data-mobile-menu-target");
  const panel = panelId ? document.getElementById(panelId) : null;
  const backdropId = button.getAttribute("data-mobile-menu-backdrop");
  const backdrop = backdropId ? document.getElementById(backdropId) : null;

  if (!panel) {
    continue;
  }

  button.addEventListener("click", () => {
    const expanded = button.getAttribute("aria-expanded") === "true";

    for (const otherButton of mobileMenuButtons) {
      const otherPanelId = otherButton.getAttribute("data-mobile-menu-target");
      const otherPanel = otherPanelId ? document.getElementById(otherPanelId) : null;
      const otherBackdropId = otherButton.getAttribute("data-mobile-menu-backdrop");
      const otherBackdrop = otherBackdropId ? document.getElementById(otherBackdropId) : null;
      if (otherPanel) {
        closeMobileMenu(otherButton, otherPanel, otherBackdrop);
      }
    }

    if (!expanded) {
      openMobileMenu(button, panel, backdrop);
    }
  });

  panel.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) {
      return;
    }
    if (target.closest("a")) {
      closeMobileMenu(button, panel, backdrop);
    }
  });

  backdrop?.addEventListener("click", () => {
    closeMobileMenu(button, panel, backdrop);
  });
}

document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape") {
    return;
  }

  for (const button of mobileMenuButtons) {
    const panelId = button.getAttribute("data-mobile-menu-target");
    const panel = panelId ? document.getElementById(panelId) : null;
    const backdropId = button.getAttribute("data-mobile-menu-backdrop");
    const backdrop = backdropId ? document.getElementById(backdropId) : null;
    if (panel) {
      closeMobileMenu(button, panel, backdrop);
    }
  }
});

const tryButtons = Array.from(document.querySelectorAll(".try-in-editor-button"));

for (const button of tryButtons) {
  button.addEventListener("click", () => {
    const shell = button.closest(".code-block-shell");
    const template = shell?.querySelector(".copy-source");
    const source = template?.content?.textContent ?? template?.textContent ?? "";
    if (!source) return;

    const encoded = btoa(unescape(encodeURIComponent(source)));
    const editorBase = document.documentElement.dataset.editorBase ?? "/editor/";
    const url = new URL(editorBase, window.location.origin);
    // ?try= signals "user is just exploring a snippet"; the editor keeps
    // it in memory until the first edit instead of saving it as a draft.
    // ?content= is reserved for actually-shared links, which do save.
    url.searchParams.set("try", encoded);
    window.open(url.toString(), "_blank");
  });
}

const copyButtons = Array.from(document.querySelectorAll(".copy-code-button"));

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const shell = button.closest(".code-block-shell");
    const template = shell?.querySelector(".copy-source");
    const status = shell?.querySelector(".copy-code-status");
    const source = template?.content?.textContent ?? template?.textContent ?? "";

    if (!source) {
      return;
    }

    try {
      await navigator.clipboard.writeText(source);
      const previous = button.textContent;
      button.textContent = "Copied";
      if (status) {
        status.textContent = "Code copied to clipboard.";
      }
      window.setTimeout(() => {
        button.textContent = previous ?? "Copy";
        if (status) {
          status.textContent = "";
        }
      }, 1500);
    } catch {
      button.textContent = "Failed";
      if (status) {
        status.textContent = "Copy failed.";
      }
      window.setTimeout(() => {
        button.textContent = "Copy";
        if (status) {
          status.textContent = "";
        }
      }, 1500);
    }
  });
}
