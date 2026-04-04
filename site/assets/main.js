const navLinks = Array.from(
  document.querySelectorAll(".section-links a[href^='#'], .docs-nav nav a[href^='#']"),
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
