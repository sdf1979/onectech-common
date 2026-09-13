(function(){
  if (window.__octInit) return; window.__octInit = 1;

  // Собираем индекс: id -> tr, parentId -> [tr, ...], плюс
  // кэш ключей сортировки по каждой колонке.
  function buildIndex(tbody) {
    var childIds = new Map();
    for (var i = 0; i < tbody.rows.length; i++) {
      var tr = tbody.rows[i];
      var id = tr.dataset.id, pid = tr.dataset.parent || "";
      var arr = childIds.get(pid);
      if (!arr) { arr = []; childIds.set(pid, arr); }
      arr.push(tr);

      var tds = tr.cells;
      var keys = new Array(tds.length - 1);
      for (var j = 1; j < tds.length; j++) {
        var k = tds[j].dataset.k || "";
        var n = parseFloat(k);
        keys[j - 1] = (k === "" || isNaN(n)) ? k : n;
      }
      tr._keys = keys;
    }
    return { childIds: childIds };
  }

  // После первого рендера замеряем реальные ширины колонок
  // и переводим таблицу в table-layout:fixed с <colgroup>.
  function freezeColumnWidths(tbl) {
    if (tbl.dataset.octFrozen) return;
    if (!tbl.tHead || !tbl.tHead.rows.length) return;
    var ths = tbl.tHead.rows[0].cells;
    var n = ths.length;
    if (!n) return;

    var widths = new Array(n);
    var total = 0;
    for (var i = 0; i < n; i++) {
      var w = ths[i].getBoundingClientRect().width;
      if (!(w > 0)) return;
      widths[i] = w;
      total += w;
    }
    if (total <= 0) return;

    var cg = document.createElement("colgroup");
    for (var i = 0; i < n; i++) {
      var col = document.createElement("col");
      col.style.width = (widths[i] / total * 100).toFixed(4) + "%";
      cg.appendChild(col);
    }
    tbl.insertBefore(cg, tbl.tHead);
    tbl.style.tableLayout = "fixed";
    tbl.dataset.octFrozen = "1";
  }

  function cmp(a, b, dir) {
    var ta = typeof a, tb = typeof b;
    if (ta === "number" && tb === "number") return (a - b) * dir;
    var sa = ta === "string" ? a : String(a);
    var sb = tb === "string" ? b : String(b);
    if (sa < sb) return -dir;
    if (sa > sb) return  dir;
    return 0;
  }

  // Рекурсивно сортируем соседей на каждом уровне — дерево сохраняется.
  function reorder(tbody, idx, col, dir) {
    var ci = col - 1;
    var out = [];
    (function walk(pid) {
      var kids = idx.childIds.get(pid);
      if (!kids) return;
      if (col > 0) {
        kids.sort(function (a, b) {
          return cmp(a._keys[ci], b._keys[ci], dir);
        });
      }
      for (var i = 0; i < kids.length; i++) {
        out.push(kids[i]);
        walk(kids[i].dataset.id);
      }
    })("");

    var frag = document.createDocumentFragment();
    for (var i = 0; i < out.length; i++) frag.appendChild(out[i]);
    tbody.replaceChildren(frag);
  }

  // Ограничение видимых строк: у каждого родителя оставляем только
  // тех детей, которые в сумме по активной колонке дают >= 80 %
  // значения родителя, плюс одного следующего ребёнка.
  //
  // Считаем всегда по убыванию значений колонки — «80 % показателя»
  // имеет смысл только для самых крупных детей, независимо от того,
  // в какую сторону сейчас отсортирован столбец для отображения.
  //
  // Рекурсия идёт только по видимым детям — значит, поддеревья
  // скрытых родителей автоматически остаются невидимыми.
  function applyFilter(tbody, idx, col) {
    var ci = col - 1;
    var visible = new Set();

    function asNumber(v) {
      return typeof v === "number" ? v : Number(v);
    }

    function filterChildren(parentRow) {
      var kids = idx.childIds.get(parentRow.dataset.id);
      if (!kids || !kids.length) return;

      var pv = asNumber(parentRow._keys[ci]);

      // Сортировка по значению колонки — по убыванию.
      var sorted = kids.slice().sort(function (a, b) {
        var an = asNumber(a._keys[ci]);
        var bn = asNumber(b._keys[ci]);
        if (isFinite(an) && isFinite(bn)) return bn - an;
        var as = String(a._keys[ci]), bs = String(b._keys[ci]);
        return as < bs ? 1 : (as > bs ? -1 : 0);
      });

      // По умолчанию (нет числового значения родителя) показываем всех.
      var limit = sorted.length - 1;
      if (isFinite(pv) && pv > 0) {
        var target = 0.8 * pv;
        var sum = 0;
        for (var i = 0; i < sorted.length; i++) {
          var n = asNumber(sorted[i]._keys[ci]);
          if (isFinite(n)) sum += n;
          if (sum >= target) {
            // +1 следующая строка.
            limit = Math.min(i + 1, sorted.length - 1);
            break;
          }
        }
      }

      for (var i = 0; i < sorted.length; i++) {
        if (i <= limit) {
          visible.add(sorted[i]);
          filterChildren(sorted[i]);
        }
      }
    }

    // Корень (Итого) всегда виден, дальше фильтруем его детей.
    var roots = idx.childIds.get("") || [];
    for (var r = 0; r < roots.length; r++) {
      visible.add(roots[r]);
      filterChildren(roots[r]);
    }

    var all = tbody.rows;
    for (var i = 0; i < all.length; i++) {
      all[i].style.display = visible.has(all[i]) ? "" : "none";
    }
  }

  // Снять ограничение — показать все строки.
  function showAll(tbody) {
    var all = tbody.rows;
    for (var i = 0; i < all.length; i++) {
      all[i].style.display = "";
    }
  }

  function init() {
    var tables = document.querySelectorAll("table.oct-table");
    for (var t = 0; t < tables.length; t++) {
      (function (tbl) {
        var tbody = tbl.tBodies[0];

        // 1) Замораживаем ширины колонок ДО любых манипуляций.
        freezeColumnWidths(tbl);

        // 2) Строим индекс для сортировки.
        var idx = buildIndex(tbody);

        // 3) Стартовое состояние сортировки из data-атрибутов.
        var initialCol = parseInt(tbl.dataset.sortCol, 10);
        if (isNaN(initialCol)) initialCol = -1;
        var initialDir = parseInt(tbl.dataset.sortDir, 10);
        if (isNaN(initialDir) || initialDir === 0) initialDir = -1;
        var state = { col: initialCol, dir: initialDir };

        // 4) Свой флажок ограничения выборки для этой таблицы.
        //    Связка через id: table[data-filter] -> input#id.
        var filterCb = null;
        if (tbl.dataset.filter) {
          filterCb = document.getElementById(tbl.dataset.filter);
        }
        var filterEnabled = filterCb ? filterCb.checked : false;

        // 5) Применить текущее состояние фильтра.
        //    Без активной колонки сортировки фильтровать нечего —
        //    показываем всё (как и было до появления флажка).
        function refreshFilter() {
          if (filterEnabled && state.col > 0) {
            applyFilter(tbody, idx, state.col);
          } else {
            showAll(tbody);
          }
        }

        if (filterCb) {
          filterCb.addEventListener("change", function () {
            filterEnabled = filterCb.checked;
            refreshFilter();
          });
        }

        refreshFilter();

        // 6) Разворот длинных ячеек по клику.
        tbody.addEventListener("click", function (e) {
          var toggle = e.target.closest(".oct-toggle");
          if (!toggle) return;
          var cell = toggle.closest(".oct-cell");
          if (!cell || !cell.classList.contains("oct-clip")) return;
          cell.classList.toggle("oct-open");
        });

        // 7) Сортировка по клику на заголовок + пересчёт фильтра.
        tbl.tHead.addEventListener("click", function (e) {
          var th = e.target.closest("th.oct-sortable");
          if (!th) return;
          var col = parseInt(th.dataset.col, 10);
          var dir = (state.col === col) ? -state.dir : -1;

          var arrows = tbl.querySelectorAll("th.oct-sortable .oct-arrow");
          for (var i = 0; i < arrows.length; i++) {
            arrows[i].classList.add("oct-arrow-off");
          }
          var arrow = th.querySelector(".oct-arrow");
          if (arrow) {
            arrow.classList.remove("oct-arrow-off");
            arrow.textContent = dir === 1 ? "▲" : "▼";
          }

          state = { col: col, dir: dir };
          reorder(tbody, idx, col, dir);
          refreshFilter();
        });
      })(tables[t]);
    }
  }

  // Гарантируем, что замер ширин делаем ПОСЛЕ первой отрисовки
  // (иначе таблица может вернуть нулевые ширины).
  function boot() {
    requestAnimationFrame(function () {
      requestAnimationFrame(init);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();