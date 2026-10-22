// content.js provides the interactive navigation elements for Cogent Core
// content websites, using Vue.js. The website is either the preview shown
// while the app is loading, or a freestanding website without the
// app (core build js -htmlonly).
// It replicates the table of contents and categories trees and the page
// search of the content package. The site navigation data is in
// window.ccSite (from nav.js), and the current page data is in window.ccPage.
// Without JavaScript, the static navigation links in the page are used instead.

(function () {
  "use strict";

  const site = window.ccSite || { Name: "", Pages: [], Categories: [] };
  const page = window.ccPage || { Name: "", URL: "", TOC: [] };
  const base = document.documentElement.dataset.basePath || "";

  const iconPaths = {
    down: "M480-344 240-584l43-43 197 197 197-197 43 43-240 240Z",
    right: "M530-481 332-679l43-43 241 241-241 241-43-43 198-198Z",
    search: "M796-121 533-384q-30 26-69.959 40.5T378-329q-108.162 0-183.081-75Q120-479 120-585t75-181q75-75 181.5-75t181 75Q632-691 632-584.85 632-542 618-502q-14 40-42 75l264 262-44 44ZM377-389q81.25 0 138.125-57.5T572-585q0-81-56.875-138.5T377-781q-82.083 0-139.542 57.5Q180-666 180-585t57.458 138.5Q294.917-389 377-389Z",
    article: "M277-279h275v-60H277v60Zm0-171h406v-60H277v60Zm0-171h406v-60H277v60Zm-97 501q-24 0-42-18t-18-42v-600q0-24 18-42t42-18h600q24 0 42 18t18 42v600q0 24-18 42t-42 18H180Zm0-60h600v-600H180v600Zm0-600v600-600Z",
  };
  const icon = (name) => `<svg viewBox="0 -960 960 960" aria-hidden="true"><path d="${iconPaths[name]}"/></svg>`;

  // pageHref returns the relative link to the page with the given URL.
  const pageHref = (url) => (url === "" ? base || "./" : base + url + "/");

  const escapeHTML = (s) =>
    s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

  const kebab = (s) =>
    s.trim().toLowerCase().replace(/[^a-z0-9_]+/g, "-").replace(/^-+|-+$/g, "");

  // findTarget returns the page element for the given location hash,
  // tolerating differences in case and spacing (like the content package).
  function findTarget(hash) {
    if (!hash || hash === "#") {
      return null;
    }
    let id = hash.slice(1);
    try {
      id = decodeURIComponent(id);
    } catch (e) {}
    return document.getElementById(id) || document.getElementById(id.toLowerCase()) || document.getElementById(kebab(id));
  }

  ////////  Math

  function renderMath() {
    if (!window.renderMathInElement) {
      return;
    }
    try {
      window.renderMathInElement(document.getElementById("cc-page"), {
        delimiters: [
          { left: "\\[", right: "\\]", display: true },
          { left: "\\(", right: "\\)", display: false },
        ],
        throwOnError: false,
      });
    } catch (e) {
      console.error("error rendering math:", e);
    }
  }

  ////////  Layout: splitter and navigation drawer

  const nav = document.getElementById("cc-nav");
  const menu = document.getElementById("cc-menu");

  function setDrawer(open) {
    nav.classList.toggle("cc-nav-open", open);
    menu.setAttribute("aria-expanded", String(open));
  }

  function setupLayout() {
    menu.addEventListener("click", (e) => {
      e.stopPropagation();
      setDrawer(!nav.classList.contains("cc-nav-open"));
    });
    document.addEventListener("click", (e) => {
      if (nav.classList.contains("cc-nav-open") && !nav.contains(e.target)) {
        setDrawer(false);
      }
    });

    const root = document.documentElement;
    try {
      const w = localStorage.getItem("cc-nav-width");
      if (w) {
        root.style.setProperty("--nav-width", w);
      }
    } catch (e) {}
    const splitter = document.getElementById("cc-splitter");
    splitter.addEventListener("pointerdown", (e) => {
      e.preventDefault();
      splitter.setPointerCapture(e.pointerId);
      const move = (e) => {
        const left = nav.getBoundingClientRect().left;
        const w = Math.min(Math.max(e.clientX - left - 8, 0.1 * innerWidth), 0.6 * innerWidth);
        root.style.setProperty("--nav-width", w + "px");
      };
      const up = () => {
        splitter.removeEventListener("pointermove", move);
        splitter.removeEventListener("pointerup", up);
        try {
          localStorage.setItem("cc-nav-width", root.style.getPropertyValue("--nav-width"));
        } catch (e) {}
      };
      splitter.addEventListener("pointermove", move);
      splitter.addEventListener("pointerup", up);
    });
  }

  ////////  Trees

  // tocTree returns the table of contents tree for the page headings,
  // nested in the same way as in the content package.
  function tocTree() {
    if (!page.TOC || page.TOC.length === 0) {
      return null;
    }
    const root = { key: "toc", html: "<b>Contents</b>", top: true, children: [] };
    const last = {};
    for (const h of page.TOC) {
      let parent = root;
      for (let i = h.Level - 1; i >= 1; i--) {
        if (last[i]) {
          parent = last[i];
          break;
        }
      }
      const node = { key: "toc:" + h.ID, html: h.Text, href: "#" + h.ID, children: [] };
      parent.children.push(node);
      last[h.Level] = node;
    }
    return root;
  }

  // categoriesTree returns the categories tree for the site, with the
  // category of the current page open and the current page selected.
  function categoriesTree() {
    if (!site.Categories || site.Categories.length === 0) {
      return null;
    }
    const root = { key: "cats", html: "<b>Categories</b>", href: pageHref(""), children: [] };
    for (const cat of site.Categories) {
      const cn = {
        key: "cat:" + cat.Name,
        html: escapeHTML(cat.Name),
        href: cat.HasPage ? pageHref(cat.URL) : null,
        closed: true,
        children: [],
      };
      if (page.Name === cat.Name) {
        root.selected = cn.key;
      }
      for (const pg of cat.Pages || []) {
        const pn = { key: "page:" + pg.URL, html: escapeHTML(pg.Name), href: pageHref(pg.URL), children: [] };
        if (pg.URL === page.URL) {
          root.selected = pn.key;
          cn.closed = false;
        }
        cn.children.push(pn);
      }
      root.children.push(cn);
    }
    return root;
  }

  const TreeNode = {
    name: "TreeNode",
    props: { node: Object, depth: Number, root: Object },
    data() {
      return { open: !this.node.closed };
    },
    computed: {
      hasChildren() {
        return this.node.children.length > 0;
      },
      selected() {
        return this.root.selected === this.node.key;
      },
      branchIcon() {
        return this.hasChildren ? icon(this.open ? "down" : "right") : "";
      },
    },
    methods: {
      toggle() {
        if (this.hasChildren) {
          this.open = !this.open;
        }
      },
      click(e) {
        if (e.ctrlKey || e.metaKey || e.shiftKey) {
          return; // open in new tab or window
        }
        if (this.node.top) { // root of table of contents: go to top of page
          e.preventDefault();
          document.getElementById("cc-page").scrollTo({ top: 0 });
          history.replaceState(null, "", location.pathname + location.search);
        } else if (!this.node.href) {
          this.toggle();
        }
        this.root.selected = this.node.key;
        if (this.node.href && this.node.href[0] === "#") {
          setDrawer(false);
        }
      },
    },
    template: `
      <div class="cc-tree-node" role="none">
        <component :is="node.href ? 'a' : 'div'" class="cc-tree-row" :class="{'cc-selected': selected}"
            :href="node.href || undefined" :style="{marginLeft: 'calc(' + depth + ' * (1em + 4px))'}"
            role="treeitem" :aria-selected="String(selected)" :aria-expanded="hasChildren ? String(open) : undefined"
            @click="click">
          <span class="cc-tree-branch" @click.stop.prevent="toggle" v-html="branchIcon"></span>
          <span class="cc-tree-label" v-html="node.html"></span>
        </component>
        <div v-if="hasChildren && open" role="group">
          <tree-node v-for="child in node.children" :key="child.key" :node="child" :depth="depth + 1" :root="root"></tree-node>
        </div>
      </div>`,
  };

  function mountNav() {
    const roots = Vue.reactive([tocTree(), categoriesTree()].filter((r) => r));
    const toc = roots.find((r) => r.key === "toc");
    // selectHash selects the table of contents node for the current location hash.
    const selectHash = () => {
      const el = findTarget(location.hash);
      if (el && toc) {
        toc.selected = "toc:" + el.id;
      }
    };
    selectHash();
    window.addEventListener("hashchange", selectHash);
    Vue.createApp({
      components: { TreeNode },
      setup() {
        return { roots };
      },
      template: `
        <div v-for="r in roots" :key="r.key" class="cc-tree-root" role="tree">
          <tree-node :node="r" :depth="0" :root="r"></tree-node>
        </div>`,
    }).mount(nav);
  }

  ////////  Search

  function mountSearch() {
    Vue.createApp({
      data() {
        return { visible: false, query: "", index: 0, siteName: site.Name };
      },
      computed: {
        results() {
          const q = this.query.trim().toLowerCase();
          const pages = (site.Pages || []).filter((p) => p.Name.toLowerCase().includes(q));
          if (q) { // prefix matches first
            pages.sort((a, b) => b.Name.toLowerCase().startsWith(q) - a.Name.toLowerCase().startsWith(q));
          }
          return pages;
        },
      },
      watch: {
        query() {
          this.index = 0;
        },
      },
      mounted() {
        document.addEventListener("keydown", (e) => {
          const t = e.target;
          const editing = t.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(t.tagName);
          if ((e.key === "/" && !editing) || (e.key === "k" && (e.ctrlKey || e.metaKey))) {
            e.preventDefault();
            this.show();
          }
        });
      },
      methods: {
        href: pageHref,
        show() {
          this.visible = true;
          this.query = "";
          this.$nextTick(() => this.$refs.input.focus());
        },
        hide() {
          this.visible = false;
        },
        key(e) {
          const n = this.results.length;
          if (e.key === "Escape") {
            this.hide();
          } else if (e.key === "ArrowDown" && n) {
            e.preventDefault();
            this.index = (this.index + 1) % n;
            this.scrollToIndex();
          } else if (e.key === "ArrowUp" && n) {
            e.preventDefault();
            this.index = (this.index - 1 + n) % n;
            this.scrollToIndex();
          } else if (e.key === "Enter" && n) {
            location.href = pageHref(this.results[this.index].URL);
          }
        },
        scrollToIndex() {
          this.$nextTick(() => {
            const el = document.querySelector(".cc-result.cc-selected");
            if (el) {
              el.scrollIntoView({ block: "nearest" });
            }
          });
        },
      },
      template: `
        <button type="button" class="cc-search-button" title="Search for pages by name (/)" @click="show">
          ${icon("search")}<span>Search</span>
        </button>
        <teleport to=".cc-site">
          <div v-if="visible" class="cc-scrim" @mousedown.self="hide">
            <div class="cc-dialog" role="dialog" aria-modal="true" aria-label="Page search">
              <input ref="input" v-model="query" type="search" :placeholder="'Search ' + siteName"
                  aria-label="Search for pages by name" @keydown="key">
              <div class="cc-results" role="listbox">
                <a v-for="(r, i) in results" :key="r.URL" class="cc-result" :class="{'cc-selected': i === index}"
                    :href="href(r.URL)" role="option" :aria-selected="String(i === index)" @mousemove="index = i">
                  ${icon("article")}<span>{{ r.Name }}</span>
                </a>
                <div v-if="results.length === 0" class="cc-no-results">No pages found</div>
              </div>
            </div>
          </div>
        </teleport>`,
    }).mount("#cc-search");
  }

  ////////  Start

  setupLayout();
  renderMath();
  if (window.Vue) {
    mountNav();
    mountSearch();
  } else {
    console.error("Vue.js could not be loaded; using static navigation");
  }
  const target = findTarget(location.hash);
  if (target) {
    target.scrollIntoView();
  }
})();
