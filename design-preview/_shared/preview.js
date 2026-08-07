const svgSymbols="<svg width=\"0\" height=\"0\" style=\"position:absolute\" aria-hidden=\"true\">\n  <symbol id=\"i-folder\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z\"/></symbol>\n  <symbol id=\"i-gear\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/></symbol>\n  <symbol id=\"i-power\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M12 2v10\"/><path d=\"M18.4 6.6a9 9 0 1 1-12.77.04\"/></symbol>\n  <symbol id=\"i-plus\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M5 12h14\"/><path d=\"M12 5v14\"/></symbol>\n  <symbol id=\"i-play\" viewBox=\"0 0 24 24\" fill=\"currentColor\" stroke=\"none\"><path d=\"M6 3l14 9-14 9V3z\"/></symbol>\n  <symbol id=\"i-check\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"M20 6 9 17l-5-5\"/></symbol>\n  <symbol id=\"i-dots\" viewBox=\"0 0 24 24\" fill=\"currentColor\"><circle cx=\"12\" cy=\"5\" r=\"1.5\"/><circle cx=\"12\" cy=\"12\" r=\"1.5\"/><circle cx=\"12\" cy=\"19\" r=\"1.5\"/></symbol>\n  <symbol id=\"i-chev\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"m6 9 6 6 6-6\"/></symbol>\n  <symbol id=\"i-term\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path d=\"m4 17 6-6-6-6\"/><path d=\"M12 19h8\"/></symbol>\n  <symbol id=\"i-mark\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><circle cx=\"12\" cy=\"12\" r=\"9\"/><path d=\"m8 12 2.5 2.5L16 9\"/></symbol>\n</svg>";
document.body.insertAdjacentHTML('afterbegin', svgSymbols);
const TERM_OUT = `<span class="t-prompt">$</span> npm run dev
<span class="t-dim">› dev server starting…</span>

  <span class="t-key">VITE</span> v5.4.2  <span class="t-ok">ready in 312 ms</span>

  ➜  Local:   <span class="t-url">http://localhost:5173/</span>
  ➜  Network: use --host to expose

<span class="t-info">[hmr]</span> /src/App.tsx updated.
<span class="t-ok">✓</span> +13 modules transformed in 89ms`;
function appHTML(){
  return `
  <div class="stage-bar">
    <span class="tl"><i></i><i></i><i></i></span>
    <span class="win-title"></span>
  </div>
  <div class="app">
    <header class="appbar">
      <div class="brand"><span class="brand-mark"><svg viewBox="0 0 24 24"><use href="#i-mark"/></svg></span><span class="brand-name"></span></div>
      <div class="spacer"></div>
      <button class="iconbtn" title="额外信息管理"><svg viewBox="0 0 24 24"><use href="#i-folder"/></svg></button>
      <button class="iconbtn" title="设置"><svg viewBox="0 0 24 24"><use href="#i-gear"/></svg></button>
      <button class="iconbtn" title="退出"><svg viewBox="0 0 24 24"><use href="#i-power"/></svg></button>
    </header>
    <div class="body">
      <aside class="sidebar">
        <div class="side-head">
          <span class="side-label">任务与终端</span>
          <div class="side-actions">
            <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-chev"/></svg></button>
            <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-plus"/></svg></button>
          </div>
        </div>
        <nav class="tabs">
          <button class="tab">未执行 <em>3</em></button>
          <button class="tab active">执行中 <em>2</em></button>
          <button class="tab">已完成 <em>5</em></button>
        </nav>
        <ul class="tasklist">
          <!-- task 1: expanded -->
          <li class="task" style="--c:#4f46e5;border-left-color:#4f46e5">
            <div class="task-row">
              <button class="caret"><svg viewBox="0 0 24 24"><use href="#i-chev"/></svg></button>
              <div class="task-main"><div class="task-title">前端开发服务器</div><div class="task-sub">Vite · React · :5173</div></div>
              <span class="dot working"></span>
              <div class="task-actions">
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-check"/></svg></button>
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-dots"/></svg></button>
              </div>
            </div>
            <ul class="terms">
              <li class="term"><svg class="t-ic" viewBox="0 0 24 24"><use href="#i-term"/></svg><span class="t-name term-name">vite — localhost:5173</span><span class="dot working"></span></li>
              <li class="term"><svg class="t-ic" viewBox="0 0 24 24"><use href="#i-term"/></svg><span class="t-name term-name">storybook — :6006</span><span class="dot idle"></span></li>
            </ul>
          </li>
          <!-- task 2: collapsed working -->
          <li class="task collapsed" style="--c:#10b981;border-left-color:#10b981">
            <div class="task-row">
              <button class="caret"><svg viewBox="0 0 24 24"><use href="#i-chev"/></svg></button>
              <div class="task-main"><div class="task-title">后端 API 服务</div><div class="task-sub">Go · Gin · :8080</div></div>
              <span class="dot working"></span>
              <div class="task-actions">
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-check"/></svg></button>
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-dots"/></svg></button>
              </div>
            </div>
          </li>
          <!-- task 3: collapsed unread -->
          <li class="task collapsed" style="--c:#f59e0b;border-left-color:#f59e0b">
            <div class="task-row">
              <button class="caret"><svg viewBox="0 0 24 24"><use href="#i-chev"/></svg></button>
              <div class="task-main"><div class="task-title">定时数据同步</div><div class="task-sub">Cron · 每小时 · 12 分钟前</div></div>
              <span class="dot unread"></span>
              <div class="task-actions">
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-check"/></svg></button>
                <button class="iconbtn"><svg viewBox="0 0 24 24"><use href="#i-dots"/></svg></button>
              </div>
            </div>
          </li>
        </ul>
      </aside>
      <main class="pane">
        <div class="term-head">
          <span class="term-tab active"><svg style="width:13px;height:13px" viewBox="0 0 24 24"><use href="#i-term"/></svg> vite — localhost:5173</span>
          <span class="term-tab">storybook — :6006</span>
          <span class="term-meta"><span class="dot working" style="width:7px;height:7px"></span> 工作中</span>
        </div>
        <pre class="term-body">${TERM_OUT}</pre>
      </main>
    </div>
  </div>`;
}
const config=window.taskaiPreviewConfig||{};
document.querySelectorAll('.stage[data-variant]').forEach((stage)=>{stage.innerHTML=appHTML();const mock=stage.closest('.mockup');const name=mock.querySelector('.brand-name');const winTitle=mock.querySelector('.win-title');const bar=mock.querySelector('.stage-bar');if(config.brandHtml)name.innerHTML=config.brandHtml;else name.textContent=config.brandText||config.fallback||name.textContent;if(config.windowHtml)winTitle.innerHTML=config.windowHtml;else winTitle.textContent=config.windowText||config.fallback||winTitle.textContent;if(config.extraClass){const extra=document.createElement('span');extra.className=config.extraClass;extra.textContent=config.extraText;bar.appendChild(extra);}});
document.querySelectorAll('.pick').forEach((button)=>button.addEventListener('click',()=>{document.querySelectorAll('.pick').forEach((item)=>item.classList.toggle('chosen',item===button));}));
if(new URLSearchParams(location.search).get('show')){document.querySelectorAll('.theme-page-head,.theme-page-foot').forEach((element)=>{element.style.display='none';});}
