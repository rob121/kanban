(function () {
    'use strict';

    var themeSaveTimer = null;

    if (window.__kanbanAppBootstrapped) {
        return;
    }
    window.__kanbanAppBootstrapped = true;

    function onPageLoad() {
        syncThemeFromPage();
        initSortable();
        initColumnSortable();
        bindDeleteColumn();
        syncColumnDeleteButtons();
    }

    function getTheme() {
        return document.documentElement.getAttribute('data-bs-theme') || 'light';
    }

    function resolveTheme() {
        var meta = document.querySelector('meta[name="user-theme"]');
        if (meta) {
            var fromMeta = meta.getAttribute('content');
            if (fromMeta === 'light' || fromMeta === 'dark') {
                return fromMeta;
            }
        }
        var stored = localStorage.getItem('kanban-theme');
        if (stored === 'light' || stored === 'dark') {
            return stored;
        }
        return 'light';
    }

    function syncThemeIcons(theme) {
        document.querySelectorAll('.theme-icon-light').forEach(function (el) {
            el.classList.toggle('d-none', theme === 'dark');
        });
        document.querySelectorAll('.theme-icon-dark').forEach(function (el) {
            el.classList.toggle('d-none', theme !== 'dark');
        });
        document.querySelectorAll('.theme-toggle').forEach(function (btn) {
            btn.setAttribute('aria-pressed', theme === 'dark' ? 'true' : 'false');
        });
    }

    function syncThemeFromPage() {
        var theme = resolveTheme();
        document.documentElement.setAttribute('data-bs-theme', theme);
        localStorage.setItem('kanban-theme', theme);
        syncThemeIcons(theme);
    }

    function setTheme(theme) {
        if (theme !== 'light' && theme !== 'dark') {
            theme = 'light';
        }
        document.documentElement.setAttribute('data-bs-theme', theme);
        localStorage.setItem('kanban-theme', theme);
        var meta = document.querySelector('meta[name="user-theme"]');
        if (meta) {
            meta.setAttribute('content', theme);
        }
        syncThemeIcons(theme);
    }

    function persistTheme(theme) {
        if (themeSaveTimer) {
            clearTimeout(themeSaveTimer);
        }
        themeSaveTimer = setTimeout(function () {
            var body = new URLSearchParams();
            body.set('theme', theme);
            appendCSRF(body);
            fetch('/account/theme', {
                method: 'POST',
                headers: csrfHeaders('application/x-www-form-urlencoded'),
                credentials: 'same-origin',
                body: body.toString()
            }).catch(function () {});
        }, 150);
    }

    function bindThemeToggle() {
        if (window.__kanbanThemeToggleBound) {
            return;
        }
        window.__kanbanThemeToggleBound = true;
        document.addEventListener('click', function (e) {
            var btn = e.target.closest('.theme-toggle');
            if (!btn) {
                return;
            }
            e.preventDefault();
            e.stopPropagation();
            var next = getTheme() === 'dark' ? 'light' : 'dark';
            setTheme(next);
            persistTheme(next);
        }, true);
    }

    function initTheme() {
        syncThemeFromPage();
        bindThemeToggle();
    }

    function applyTheme() {
        setTheme(resolveTheme());
    }

    function csrfToken() {
        var el = document.querySelector('meta[name="csrf-token"]');
        return el ? el.getAttribute('content') : '';
    }

    function csrfHeaders(contentType) {
        var headers = { 'X-CSRF-Token': csrfToken() };
        if (contentType) {
            headers['Content-Type'] = contentType;
        }
        return headers;
    }

    function appendCSRF(body) {
        if (body instanceof URLSearchParams) {
            body.set('csrf_token', csrfToken());
            return body;
        }
        if (body instanceof FormData) {
            body.set('csrf_token', csrfToken());
            return body;
        }
        return body;
    }

    function showFormError(errorEl, message) {
        if (!errorEl) return;
        errorEl.textContent = message;
        errorEl.classList.remove('d-none');
    }

    function hideFormError(errorEl) {
        if (!errorEl) return;
        errorEl.textContent = '';
        errorEl.classList.add('d-none');
    }

    function initAddCardModal() {
        if (window.__kanbanAddCardModalBound) {
            return;
        }
        window.__kanbanAddCardModalBound = true;

        document.addEventListener('click', function (e) {
            var btn = e.target.closest('.add-card-btn');
            if (!btn) {
                return;
            }
            var categoryId = btn.getAttribute('data-category-id') || '';
            var categoryInput = document.getElementById('add-card-category-id');
            if (categoryInput) {
                categoryInput.value = categoryId;
            }
            var modal = document.getElementById('addCardModal');
            if (modal) {
                modal.dataset.categoryId = categoryId;
            }
        }, true);

        document.addEventListener('show.bs.modal', function (event) {
            if (event.target.id !== 'addCardModal') {
                return;
            }
            var trigger = event.relatedTarget;
            if (trigger && trigger.classList.contains('add-card-btn')) {
                var categoryId = trigger.getAttribute('data-category-id') || '';
                var categoryInput = document.getElementById('add-card-category-id');
                if (categoryInput) {
                    categoryInput.value = categoryId;
                }
                event.target.dataset.categoryId = categoryId;
            }
            hideFormError(document.getElementById('add-card-error'));
            var titleInput = document.getElementById('card-title');
            if (titleInput) {
                titleInput.classList.remove('is-invalid');
                titleInput.value = '';
            }
            var descInput = document.getElementById('card-description');
            if (descInput) {
                descInput.value = '';
            }
            event.target.querySelectorAll('.tag-pill-check').forEach(function (cb) {
                cb.checked = false;
            });
        });

        document.addEventListener('submit', function (e) {
            var form = e.target;
            if (!form || form.id !== 'add-card-form') {
                return;
            }
            e.preventDefault();

            var errorEl = document.getElementById('add-card-error');
            var titleInput = form.querySelector('[name="title"]');
            var submitBtn = form.querySelector('[type="submit"]');
            var board = document.querySelector('.kanban-board');
            var modal = document.getElementById('addCardModal');
            var boardId = (board && board.getAttribute('data-board-id'))
                || form.getAttribute('data-board-id') || '';
            var categoryId = (form.querySelector('[name="category_id"]') || {}).value
                || (modal && modal.dataset.categoryId) || '';

            hideFormError(errorEl);

            if (!titleInput || !titleInput.value.trim()) {
                if (titleInput) {
                    titleInput.classList.add('is-invalid');
                    titleInput.focus();
                }
                showFormError(errorEl, 'Title is required.');
                return;
            }
            titleInput.classList.remove('is-invalid');

            if (!boardId || !categoryId) {
                showFormError(errorEl, 'Choose a column using the + Add card button in that column.');
                return;
            }

            var body = new URLSearchParams();
            new FormData(form).forEach(function (value, key) {
                body.append(key, value);
            });
            body.set('board_id', boardId);
            body.set('category_id', categoryId);
            appendCSRF(body);

            if (submitBtn) {
                submitBtn.disabled = true;
            }

            fetch(form.action, {
                method: 'POST',
                headers: csrfHeaders('application/x-www-form-urlencoded'),
                body: body.toString(),
                redirect: 'follow'
            })
                .then(function (res) {
                    if (!res.ok) {
                        return res.text().then(function (body) {
                            var msg = body && body.length < 200 ? body : ('Could not add card (' + res.status + ').');
                            throw new Error(msg);
                        });
                    }
                    window.location.reload();
                })
                .catch(function (err) {
                    showFormError(errorEl, err.message || 'Could not add card.');
                })
                .finally(function () {
                    if (submitBtn) {
                        submitBtn.disabled = false;
                    }
                });
        });
    }

    function initColumnSortable() {
        if (typeof Sortable === 'undefined') return;

        var board = document.querySelector('.kanban-board');
        var canManage = board && board.getAttribute('data-can-manage-board') === 'true';
        var columnsEl = document.getElementById('kanban-columns');
        if (!columnsEl || !canManage || columnsEl._sortable) return;

        columnsEl._sortable = Sortable.create(columnsEl, {
            animation: 150,
            handle: '.column-drag-handle',
            draggable: '.kanban-column',
            ghostClass: 'sortable-ghost',
            onEnd: function (evt) {
                var col = evt.item;
                var categoryId = col.getAttribute('data-category-id');
                var position = evt.newIndex;
                var body = new URLSearchParams();
                body.set('position', String(position));
                appendCSRF(body);

                fetch('/categories/' + categoryId + '/move', {
                    method: 'POST',
                    headers: csrfHeaders('application/x-www-form-urlencoded'),
                    body: body.toString()
                }).then(function (res) {
                    if (!res.ok) window.location.reload();
                });
            }
        });
    }

    function bindDeleteColumn() {
        document.querySelectorAll('.delete-column-btn').forEach(function (btn) {
            if (btn.dataset.bound === '1') return;
            btn.dataset.bound = '1';
            btn.addEventListener('click', function () {
                var categoryId = btn.getAttribute('data-category-id');
                var col = btn.closest('.kanban-column');
                var cardCount = col ? col.querySelectorAll('.kanban-card').length : 0;
                if (cardCount > 0) {
                    alert('Move or archive all cards from this column before deleting it.');
                    return;
                }
                if (!confirm('Delete this column?')) return;

                fetch('/categories/' + categoryId + '/delete', {
                    method: 'POST',
                    headers: csrfHeaders()
                }).then(function (res) {
                    if (res.ok) {
                        window.location.reload();
                        return;
                    }
                    return res.text().then(function (t) {
                        alert(t || 'Could not delete column.');
                    });
                });
            });
        });
    }

    function initSortable() {
        if (typeof Sortable === 'undefined') return;

        var board = document.querySelector('.kanban-board');
        var canMove = !board || board.getAttribute('data-can-move') !== 'false';

        document.querySelectorAll('.kanban-cards').forEach(function (el) {
            if (el._sortable) {
                el._sortable.destroy();
                el._sortable = null;
            }
            el._sortable = Sortable.create(el, {
                group: 'kanban',
                animation: 150,
                handle: '.card-drag-handle',
                ghostClass: 'sortable-ghost',
                dragClass: 'sortable-drag',
                draggable: '.kanban-card',
                disabled: !canMove,
                onEnd: function (evt) {
                    var card = evt.item;
                    var cardId = card.getAttribute('data-card-id');
                    var categoryId = evt.to.getAttribute('data-category-id');
                    var position = evt.newIndex;

                    var body = new URLSearchParams();
                    body.set('category_id', categoryId);
                    body.set('position', String(position));
                    appendCSRF(body);

                    fetch('/cards/' + cardId + '/move', {
                        method: 'POST',
                        headers: csrfHeaders('application/x-www-form-urlencoded'),
                        body: body.toString()
                    }).then(function (res) {
                        if (!res.ok) console.error('Move failed');
                        updateColumnCounts();
                    });
                }
            });
        });
    }

    function updateColumnCounts() {
        document.querySelectorAll('.kanban-column').forEach(function (col) {
            var count = col.querySelectorAll('.kanban-card').length;
            var badge = col.querySelector('.column-count');
            if (badge) badge.textContent = String(count);
        });
        syncColumnDeleteButtons();
    }

    function syncColumnDeleteButtons() {
        var board = document.querySelector('.kanban-board');
        if (!board || board.getAttribute('data-can-delete-columns') !== 'true') {
            return;
        }
        document.querySelectorAll('.kanban-column').forEach(function (col) {
            var header = col.querySelector('.kanban-column-header');
            if (!header) {
                return;
            }
            var categoryId = col.getAttribute('data-category-id');
            var count = col.querySelectorAll('.kanban-card').length;
            var btn = header.querySelector('.delete-column-btn');
            if (count === 0) {
                if (!btn) {
                    btn = document.createElement('button');
                    btn.type = 'button';
                    btn.className = 'btn btn-sm btn-link text-danger p-0 delete-column-btn';
                    btn.setAttribute('data-category-id', categoryId);
                    btn.title = 'Delete column';
                    btn.setAttribute('aria-label', 'Delete column');
                    btn.innerHTML = '&times;';
                    header.appendChild(btn);
                }
            } else if (btn) {
                btn.remove();
            }
        });
        bindDeleteColumn();
    }

    function replaceKanbanCard(cardId, html) {
        var existing = document.getElementById('card-' + cardId);
        if (!existing || !html) {
            return;
        }
        existing.outerHTML = html.trim();
        initSortable();
    }

    function fetchKanbanCard(cardId) {
        return fetch('/cards/' + cardId + '?partial=1&view=card', {
            headers: {
                'X-Partial': '1',
                'Accept': 'text/html'
            },
            credentials: 'same-origin'
        }).then(function (res) {
            if (!res.ok) {
                throw new Error('Could not refresh card');
            }
            return res.text();
        });
    }

    function refreshKanbanCard(cardId) {
        return fetchKanbanCard(cardId).then(function (html) {
            replaceKanbanCard(cardId, html);
        });
    }

    function setAttachmentDropzoneBusy(zone, busy) {
        if (!zone) {
            return;
        }
        var idle = zone.querySelector('.attachment-dropzone-idle');
        var busyEl = zone.querySelector('.attachment-dropzone-busy');
        zone.classList.toggle('is-uploading', busy);
        if (idle) {
            idle.classList.toggle('d-none', busy);
        }
        if (busyEl) {
            busyEl.classList.toggle('d-none', !busy);
        }
    }

    function uploadAttachmentFile(form, file) {
        var zone = form.querySelector('.attachment-dropzone');
        var uploadError = form.querySelector('#attachment-form-error');
        if (!file) {
            return Promise.resolve();
        }
        if (uploadError) {
            uploadError.classList.add('d-none');
            uploadError.textContent = '';
        }
        setAttachmentDropzoneBusy(zone, true);

        var uploadBody = new FormData();
        uploadBody.append('file', file);
        appendCSRF(uploadBody);

        return fetch(form.action, {
            method: 'POST',
            headers: {
                'X-Partial': '1',
                'X-CSRF-Token': csrfToken(),
                'Accept': 'text/html'
            },
            credentials: 'same-origin',
            body: uploadBody
        })
            .then(function (res) {
                if (!res.ok) {
                    return res.text().then(function (msg) {
                        throw new Error(msg && msg.length < 200 ? msg : 'Could not upload file.');
                    });
                }
                return res.text();
            })
            .then(function (html) {
                var list = document.getElementById('attachments-list');
                if (list) {
                    var empty = document.getElementById('attachments-empty');
                    if (empty) {
                        empty.remove();
                    }
                    list.insertAdjacentHTML('beforeend', html);
                }
                var countEl = document.getElementById('attachments-count');
                if (countEl) {
                    countEl.textContent = String(list ? list.querySelectorAll('.attachment-item').length : 0);
                }
                var input = form.querySelector('.attachment-dropzone-input');
                if (input) {
                    input.value = '';
                }
            })
            .catch(function (err) {
                if (uploadError) {
                    uploadError.textContent = err.message || 'Could not upload file.';
                    uploadError.classList.remove('d-none');
                }
            })
            .finally(function () {
                setAttachmentDropzoneBusy(zone, false);
            });
    }

    function bindAttachmentDropzone() {
        if (window.__kanbanAttachmentDropzoneBound) {
            return;
        }
        window.__kanbanAttachmentDropzoneBound = true;

        document.addEventListener('dragover', function (e) {
            var zone = e.target.closest('.attachment-dropzone');
            if (!zone || zone.classList.contains('is-uploading')) {
                return;
            }
            e.preventDefault();
            if (e.dataTransfer) {
                e.dataTransfer.dropEffect = 'copy';
            }
            zone.classList.add('is-dragover');
        });

        document.addEventListener('dragleave', function (e) {
            var zone = e.target.closest('.attachment-dropzone');
            if (!zone) {
                return;
            }
            if (!zone.contains(e.relatedTarget)) {
                zone.classList.remove('is-dragover');
            }
        });

        document.addEventListener('dragend', function () {
            document.querySelectorAll('.attachment-dropzone.is-dragover').forEach(function (zone) {
                zone.classList.remove('is-dragover');
            });
        });

        document.addEventListener('drop', function (e) {
            var zone = e.target.closest('.attachment-dropzone');
            document.querySelectorAll('.attachment-dropzone.is-dragover').forEach(function (z) {
                z.classList.remove('is-dragover');
            });
            if (!zone || zone.classList.contains('is-uploading')) {
                return;
            }
            e.preventDefault();
            var form = zone.closest('.attachment-form');
            var file = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
            if (form && file) {
                uploadAttachmentFile(form, file);
            }
        });

        document.addEventListener('click', function (e) {
            var zone = e.target.closest('.attachment-dropzone');
            if (!zone || zone.classList.contains('is-uploading')) {
                return;
            }
            var input = zone.querySelector('.attachment-dropzone-input');
            if (input) {
                input.click();
            }
        });

        document.addEventListener('keydown', function (e) {
            var zone = e.target.closest('.attachment-dropzone');
            if (!zone || zone.classList.contains('is-uploading')) {
                return;
            }
            if (e.key !== 'Enter' && e.key !== ' ') {
                return;
            }
            e.preventDefault();
            var input = zone.querySelector('.attachment-dropzone-input');
            if (input) {
                input.click();
            }
        });

        document.addEventListener('change', function (e) {
            if (!e.target.classList.contains('attachment-dropzone-input')) {
                return;
            }
            var form = e.target.closest('.attachment-form');
            var file = e.target.files && e.target.files[0];
            if (form && file) {
                uploadAttachmentFile(form, file);
            }
        });

        document.addEventListener('submit', function (e) {
            if (e.target.classList.contains('attachment-form')) {
                e.preventDefault();
            }
        });
    }

    function openCardDetail(cardId, commentsPage) {
        var body = document.getElementById('card-detail-body');
        var offcanvasEl = document.getElementById('cardDetail');
        if (!body || !offcanvasEl || !cardId) {
            return;
        }

        body.innerHTML = '<div class="p-4 text-muted">Loading…</div>';
        var oc = window.bootstrap.Offcanvas.getOrCreateInstance(offcanvasEl);
        oc.show();

        var url = '/cards/' + cardId + '?partial=1';
        if (commentsPage) {
            url += '&comments_page=' + encodeURIComponent(commentsPage);
        }

        fetch(url, {
            headers: {
                'Turbo-Frame': 'card-detail',
                'X-Partial': '1',
                'Accept': 'text/html'
            },
            credentials: 'same-origin'
        })
            .then(function (res) {
                if (!res.ok) {
                    throw new Error('Could not load card');
                }
                return res.text();
            })
            .then(function (html) {
                body.innerHTML = html;
                var titleEl = document.getElementById('cardDetailLabel');
                var titleNode = body.querySelector('[data-card-title]');
                if (titleEl && titleNode) {
                    titleEl.textContent = titleNode.getAttribute('data-card-title') || 'Card details';
                }
            })
            .catch(function () {
                body.innerHTML = '<p class="text-danger mb-0">Could not load card details. Please try again.</p>';
            });
    }

    function bindCardInteractions() {
        if (window.__kanbanCardInteractionBound) {
            return;
        }
        window.__kanbanCardInteractionBound = true;

        document.addEventListener('click', function (e) {
            if (e.target.closest('.comments-page-btn')) {
                var pageBtn = e.target.closest('.comments-page-btn');
                var panel = pageBtn.closest('#card-detail-panel');
                if (!panel) {
                    return;
                }
                e.preventDefault();
                openCardDetail(panel.getAttribute('data-card-id'), pageBtn.getAttribute('data-comments-page'));
                return;
            }
            if (e.target.closest('.card-detail-edit-btn')) {
                var panel = e.target.closest('#card-detail-panel');
                if (!panel) {
                    return;
                }
                panel.classList.add('is-editing');
                var view = panel.querySelector('#card-detail-view');
                var form = panel.querySelector('#card-edit-form');
                if (view) {
                    view.classList.add('d-none');
                }
                if (form) {
                    form.classList.remove('d-none');
                }
                return;
            }
            if (e.target.closest('.card-detail-cancel-btn')) {
                var panel = e.target.closest('#card-detail-panel');
                if (!panel) {
                    return;
                }
                panel.classList.remove('is-editing');
                var view = panel.querySelector('#card-detail-view');
                var form = panel.querySelector('#card-edit-form');
                if (view) {
                    view.classList.remove('d-none');
                }
                if (form) {
                    form.classList.add('d-none');
                }
                return;
            }
            if (e.target.closest('.card-drag-handle')) {
                return;
            }
            var card = e.target.closest('.kanban-card');
            if (!card) {
                return;
            }
            openCardDetail(card.getAttribute('data-card-id'));
        });

        document.addEventListener('keydown', function (e) {
            if (e.key !== 'Enter' && e.key !== ' ') {
                return;
            }
            var card = e.target.closest('.kanban-card');
            if (!card || e.target.closest('.card-drag-handle')) {
                return;
            }
            e.preventDefault();
            openCardDetail(card.getAttribute('data-card-id'));
        });

        document.addEventListener('submit', function (e) {
            var form = e.target;
            if (form.id === 'card-edit-form') {
                e.preventDefault();
                var body = new URLSearchParams();
                new FormData(form).forEach(function (value, key) {
                    body.append(key, value);
                });
                appendCSRF(body);
                var panel = form.closest('#card-detail-panel');
                var cardId = panel && panel.getAttribute('data-card-id');
                var submitBtns = form.querySelectorAll('[type="submit"]');
                submitBtns.forEach(function (btn) { btn.disabled = true; });
                fetch(form.action, {
                    method: 'POST',
                    headers: Object.assign(csrfHeaders('application/x-www-form-urlencoded'), {
                        'X-Partial': '1',
                        'Accept': 'text/html'
                    }),
                    credentials: 'same-origin',
                    body: body.toString()
                }).then(function (res) {
                    if (!res.ok) {
                        throw new Error('Could not save card');
                    }
                    return res.text();
                }).then(function (html) {
                    if (cardId) {
                        replaceKanbanCard(cardId, html);
                        openCardDetail(cardId);
                    }
                }).catch(function () {
                    alert('Could not save card. Please try again.');
                }).finally(function () {
                    submitBtns.forEach(function (btn) { btn.disabled = false; });
                });
                return;
            }
            if (form.classList.contains('comment-form')) {
                e.preventDefault();
                var commentBody = new URLSearchParams();
                new FormData(form).forEach(function (value, key) {
                    commentBody.append(key, value);
                });
                appendCSRF(commentBody);
                var errorEl = form.querySelector('#comment-form-error');
                var submitBtn = form.querySelector('[type="submit"]');
                if (errorEl) {
                    errorEl.classList.add('d-none');
                    errorEl.textContent = '';
                }
                if (submitBtn) {
                    submitBtn.disabled = true;
                }
                fetch(form.action, {
                    method: 'POST',
                    headers: Object.assign(csrfHeaders('application/x-www-form-urlencoded'), {
                        'Turbo-Frame': 'comment',
                        'X-Partial': '1',
                        'Accept': 'text/html'
                    }),
                    credentials: 'same-origin',
                    body: commentBody.toString()
                })
                    .then(function (res) {
                        if (!res.ok) {
                            return res.text().then(function (msg) {
                                throw new Error(msg && msg.length < 200 ? msg : 'Could not post comment.');
                            });
                        }
                        return res.text();
                    })
                    .then(function () {
                        var panel = form.closest('#card-detail-panel');
                        var cardId = panel && panel.getAttribute('data-card-id');
                        var countEl = document.getElementById('comments-count');
                        var total = countEl ? parseInt(countEl.textContent, 10) || 0 : 0;
                        var lastPage = Math.max(1, Math.ceil((total + 1) / 5));
                        if (cardId) {
                            refreshKanbanCard(cardId).finally(function () {
                                openCardDetail(cardId, lastPage);
                            });
                        }
                        form.reset();
                    })
                    .catch(function (err) {
                        if (errorEl) {
                            errorEl.textContent = err.message || 'Could not post comment.';
                            errorEl.classList.remove('d-none');
                        }
                    })
                    .finally(function () {
                        if (submitBtn) {
                            submitBtn.disabled = false;
                        }
                    });
            }
            if (form.classList.contains('attachment-form')) {
                e.preventDefault();
                return;
            }
        });

        document.addEventListener('click', function (e) {
            var delBtn = e.target.closest('.attachment-delete-btn');
            if (delBtn) {
                e.preventDefault();
                if (!confirm('Remove this attachment?')) {
                    return;
                }
                var attachmentId = delBtn.getAttribute('data-attachment-id');
                var body = new URLSearchParams();
                appendCSRF(body);
                fetch('/attachments/' + attachmentId + '/delete', {
                    method: 'POST',
                    headers: Object.assign(csrfHeaders('application/x-www-form-urlencoded'), {
                        'X-Partial': '1'
                    }),
                    credentials: 'same-origin',
                    body: body.toString()
                }).then(function (res) {
                    if (!res.ok) {
                        throw new Error('Could not remove attachment');
                    }
                    var item = document.getElementById('attachment-' + attachmentId);
                    if (item) {
                        item.remove();
                    }
                    var list = document.getElementById('attachments-list');
                    var countEl = document.getElementById('attachments-count');
                    var remaining = list ? list.querySelectorAll('.attachment-item').length : 0;
                    if (countEl) {
                        countEl.textContent = String(remaining);
                    }
                    if (list && remaining === 0 && !document.getElementById('attachments-empty')) {
                        list.innerHTML = '<li class="card-detail-empty mb-0" id="attachments-empty">No attachments yet.</li>';
                    }
                }).catch(function () {
                    alert('Could not remove attachment.');
                });
                return;
            }
            var btn = e.target.closest('.archive-card-btn');
            if (!btn) {
                return;
            }
            if (!confirm('Archive this card?')) {
                return;
            }
            fetch('/cards/' + btn.getAttribute('data-card-id') + '/archive', {
                method: 'POST',
                headers: csrfHeaders()
            }).then(function () { location.reload(); });
        });
    }

    function initApp() {
        initTheme();
        initAddCardModal();
        bindCardInteractions();
        bindAttachmentDropzone();
        onPageLoad();
    }

    document.addEventListener('DOMContentLoaded', initApp);
    document.addEventListener('turbo:load', onPageLoad);
    document.addEventListener('turbo:before-cache', function () {
        document.documentElement.setAttribute('data-bs-theme', getTheme());
    });
})();
