(function() {
    var saved = localStorage.getItem('theme');
    var isLight = saved ? saved === 'light' : window.matchMedia('(prefers-color-scheme: light)').matches;
    if (isLight) document.documentElement.classList.add('light');
})();

function toggleTheme() {
    document.documentElement.classList.toggle('light');
    var isLight = document.documentElement.classList.contains('light');
    localStorage.setItem('theme', isLight ? 'light' : 'dark');
    updateToggleIcon();
}

function updateToggleIcon() {
    var btn = document.querySelector('.theme-toggle');
    if (!btn) return;
    var isLight = document.documentElement.classList.contains('light');
    btn.textContent = isLight ? '\u2600\uFE0F' : '\uD83C\uDF19';
}

document.addEventListener('DOMContentLoaded', function() {
    updateToggleIcon();
});

window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', function(e) {
    if (!localStorage.getItem('theme')) {
        document.documentElement.classList.toggle('light', e.matches);
        updateToggleIcon();
    }
});
