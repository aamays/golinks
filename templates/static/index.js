async function addLink(e) {
    e.preventDefault();
    const form = e.target;
    const phrase = form.phrase.value.trim();
    const url = form.url.value.trim();
    const res = await fetch('/_/api/links', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({phrase, url})
    });
    if (res.ok) location.reload();
    else alert(await res.text());
}

async function deleteLink(phrase) {
    if (!confirm('Delete go/' + phrase + '?')) return;
    const res = await fetch('/_/api/links/' + encodeURIComponent(phrase), {method: 'DELETE'});
    if (res.ok) location.reload();
    else alert(await res.text());
}

function editLink(btn) {
    const row = btn.closest('tr');
    const urlCell = row.querySelector('.url-cell');
    const currentUrl = urlCell.querySelector('.url-text').textContent;
    const phrase = row.dataset.phrase;

    urlCell.innerHTML = '<input class="edit-input" value="' + currentUrl + '">';
    const input = urlCell.querySelector('input');
    input.focus();

    const actions = row.querySelector('.actions');
    actions.innerHTML = '<button class="btn-save" onclick="saveLink(\'' + phrase + '\', this)">Save</button>';
}

async function saveLink(phrase, btn) {
    const row = btn.closest('tr');
    const newUrl = row.querySelector('.edit-input').value.trim();
    const res = await fetch('/_/api/links/' + encodeURIComponent(phrase), {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({url: newUrl})
    });
    if (res.ok) location.reload();
    else alert(await res.text());
}
