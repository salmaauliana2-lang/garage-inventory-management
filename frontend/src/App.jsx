import React, {createContext, useContext, useEffect, useState} from 'react'
import {Routes,Route,Navigate,NavLink,useNavigate,useLocation} from 'react-router-dom'
import {LayoutDashboard,Boxes,Wrench,ReceiptText,LogOut,Search,Plus,AlertTriangle,CheckCircle2,Clock3,Menu,PackagePlus,CalendarDays,Cog,RotateCw} from 'lucide-react'
import {ResponsiveContainer,AreaChart,Area,BarChart,Bar,XAxis,YAxis,Tooltip,CartesianGrid} from 'recharts'
import api from './api'

const AuthContext=createContext(null)
const money=n=>new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(n||0)
const todayISO=()=>{const d=new Date();return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,10)}
const movementTypes={restock:'Barang masuk',sale:'Penjualan',service:'Service',correction:'Koreksi'}

const ToastContext=createContext(()=>{})
const toastIcons={success:<CheckCircle2 size={16}/>,error:<AlertTriangle size={16}/>,info:<Clock3 size={16}/>}
function ToastProvider({children}) {
 const [items,setItems]=useState([])
 const push=(message,type='success')=>{
  const id=`${Date.now()}-${Math.random()}`
  setItems(list=>[...list,{id,message,type}])
  setTimeout(()=>setItems(list=>list.filter(t=>t.id!==id)),3400)
 }
 return <ToastContext.Provider value={push}>
  {children}
  <div className="toast-stack" aria-live="polite">{items.map(t=><div key={t.id} className={`toast ${t.type}`}>{toastIcons[t.type]||toastIcons.info}<span>{t.message}</span></div>)}</div>
 </ToastContext.Provider>
}
const useToast=()=>useContext(ToastContext)

function useCountUp(value) {
 const [current,setCurrent]=useState(0)
 useEffect(()=>{
  if(typeof value!=='number'||!isFinite(value)){setCurrent(value);return}
  let frame
  const start=performance.now()
  const tick=now=>{
   const p=Math.min(1,(now-start)/700)
   setCurrent(Math.round(value*(1-Math.pow(1-p,3))))
   if(p<1)frame=requestAnimationFrame(tick)
  }
  frame=requestAnimationFrame(tick)
  return()=>cancelAnimationFrame(frame)
 },[value])
 return current
}
function CountUp({value,format}) {
 const current=useCountUp(typeof value==='number'?value:0)
 if(typeof value!=='number')return <>{value??'—'}</>
 return <>{format?format(current):current}</>
}

function AuthProvider({children}) {
 const [user,setUser]=useState(()=>{try{return JSON.parse(localStorage.getItem('garage_user'))}catch{return null}})
 const login=async(email,password)=>{const {data}=await api.post('/auth/login',{email,password});localStorage.setItem('garage_token',data.token);localStorage.setItem('garage_user',JSON.stringify(data.user));setUser(data.user)}
 const register=async(name,email,password)=>{await api.post('/auth/register',{name,email,password})}
 const logout=async()=>{try{await api.post('/auth/logout')}catch{}localStorage.removeItem('garage_token');localStorage.removeItem('garage_user');setUser(null)}
 return <AuthContext.Provider value={{user,login,register,logout}}>{children}</AuthContext.Provider>
}
const useAuth=()=>useContext(AuthContext)

function Login(){
 const {login,register}=useAuth()
 const [mode,setMode]=useState('login')
 const [name,setName]=useState('')
 const [email,setEmail]=useState('manager@garage.local')
 const [password,setPassword]=useState('Garage123!')
 const [error,setError]=useState('')
 const [busy,setBusy]=useState(false)
 const nav=useNavigate()
 async function submit(e){
  e.preventDefault();setError('');setBusy(true)
  try{
   if(mode==='register'){await register(name,email,password);await login(email,password)}else await login(email,password)
   nav('/')
  }catch(err){setError(err.response?.data?.error||'Tidak dapat terhubung ke server')}finally{setBusy(false)}
 }
 return <div className="login-page">
  <div className="login-art">
   <Cog className="login-gear g1" size={280} strokeWidth={1}/>
   <Cog className="login-gear g2" size={150} strokeWidth={1}/>
   <div className="login-tach" aria-hidden="true"/>
   <i className="login-needle" aria-hidden="true"/>
   <div className="brand-mark">G<span>F</span></div>
   <p className="eyebrow">WORKSHOP OPERATIONS</p>
   <h1>Semua sparepart.<br/>Satu kendali.</h1>
   <p className="muted-light">Inventory, penjualan, dan progress service dalam satu workspace.</p>
   <div className="login-metric"><span>GARAGEFLOW</span><span>BUILT FOR YOUR WORKSHOP ↗</span></div>
  </div>
  <form className="login-card" onSubmit={submit}>
   <div className="brand-mark small">G<span>F</span></div>
   <p className="eyebrow">{mode==='login'?'SELAMAT DATANG KEMBALI':'BUAT AKUN BARU'}</p>
   <h2>{mode==='login'?'Masuk ke workspace':'Daftar akun'}</h2>
   <p className="subtle">Kelola operasional bengkel tanpa kehilangan jejak.</p>
   {mode==='register'&&<label>Nama lengkap<input required value={name} onChange={e=>setName(e.target.value)} placeholder="Nama Anda"/></label>}
   <label>Email<input type="email" required value={email} onChange={e=>setEmail(e.target.value)} placeholder="nama@bengkel.com"/></label>
   <label>Password<input type="password" minLength="8" required value={password} onChange={e=>setPassword(e.target.value)}/></label>
   {error&&<div className="error">{error}</div>}
   <button className="primary full" disabled={busy}>{busy?'Memproses…':mode==='login'?'Masuk →':'Buat akun →'}</button>
   <p className="switch">{mode==='login'?'Belum punya akun?':'Sudah punya akun?'} <button type="button" className="text-button" onClick={()=>{setMode(mode==='login'?'register':'login');setError('')}}>{mode==='login'?'Daftar':'Masuk'}</button></p>
   {mode==='login'&&<div className="demo-hint">Demo: manager@garage.local / Garage123!</div>}
  </form>
 </div>
}

function Shell({children}){
 const {user,logout}=useAuth()
 const loc=useLocation()
 const [mobile,setMobile]=useState(false)
 return <div className="app-shell">
  <aside className={`sidebar ${mobile?'open':''}`}>
   <Cog className="side-gear" size={340} strokeWidth={0.7} aria-hidden="true"/>
   <div className="brand"><div className="brand-mark">G<span>F</span></div><div><b>garageflow</b><small>WORKSHOP OS</small></div></div>
   <div className="nav-caption">WORKSPACE</div>
   <nav>
    <NavLink to="/" end><LayoutDashboard size={18}/> Overview</NavLink>
    <NavLink to="/parts"><Boxes size={18}/> Inventory</NavLink>
    <NavLink to="/services"><Wrench size={18}/> Service board</NavLink>
    <NavLink to="/sales"><ReceiptText size={18}/> Penjualan</NavLink>
   </nav>
   <div className="sidebar-bottom">
    <div className="online-dot"/> Sistem operasional <span className="online-label">ONLINE</span>
    <div className="profile"><div className="avatar">{user?.name?.slice(0,1)||'U'}</div><div className="profile-copy"><b>{user?.name}</b><small>{user?.role==='manager'?'Pengelola':'Montir'}</small></div><button title="Logout" className="icon-btn" onClick={logout}><LogOut size={17}/></button></div>
   </div>
  </aside>
  <main className="main">
   <header className="topbar">
    <button className="icon-btn mobile-menu" onClick={()=>setMobile(!mobile)}><Menu/></button>
    <div className="breadcrumb">Workspace <span>/</span> <b>Garage management</b></div>
    <div className="top-right">
     <span className="engine-chip"><Cog size={13}/> GARAGE OPEN</span>
     <span className="today">{new Date().toLocaleDateString('id-ID',{weekday:'short',day:'2-digit',month:'short',year:'numeric'})}</span>
     <div className="avatar">{user?.name?.slice(0,1)||'U'}</div>
    </div>
   </header>
   <div className="content" key={loc.pathname}>{children}</div>
  </main>
 </div>
}

function PageTitle({eyebrow,title,desc,action,no}){
 return <div className="page-title">
  <span className="page-no" aria-hidden="true">{no}</span>
  <div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p className="subtle">{desc}</p></div>
  {action}
 </div>
}

function RefreshButton({busy,onClick}){
 return <button className="ghost-btn refresh" onClick={onClick} disabled={busy} title="Muat ulang data"><RotateCw size={15} className={busy?'spin-icon':''}/> Refresh</button>
}

function Dashboard(){
 const [data,setData]=useState(null)
 const [err,setErr]=useState('')
 const [busy,setBusy]=useState(false)
 const load=()=>{setBusy(true);api.get('/dashboard').then(r=>{setData(r.data);setErr('')}).catch(e=>setErr(e.response?.data?.error||'Gagal memuat dashboard')).finally(()=>setBusy(false))}
 useEffect(()=>{load()},[])
 const stats=data?.stats||{}
 const nav=useNavigate()
 const dayName=new Date().toLocaleDateString('id-ID',{weekday:'long'}).toUpperCase()
 const low=(data?.low_stock_parts||[]).map(p=>`${p.name} · sisa ${p.stock} pcs`)
 const tickerItems=low.length?low:['Semua stok aman','Tidak ada item kritis hari ini','Katalog siap dijual']
 return <>
  <PageTitle eyebrow={`${dayName} · WORKSHOP PULSE`} title="Overview" no="01" desc="Ringkasan kondisi bengkel hari ini." action={<div className="title-actions"><span className="status-pill"><span/> Live workspace</span><RefreshButton busy={busy} onClick={load}/></div>}/>
  {err&&<div className="error">{err}</div>}
  <div className="ticker" title="Klik untuk membuka inventory stok menipis" role="button" tabIndex={0} onClick={()=>nav('/parts?low_stock=true')} onKeyDown={e=>{if(e.key==='Enter')nav('/parts?low_stock=true')}}>
   <span className="ticker-label"><AlertTriangle size={13}/> HEADS UP</span>
   <div className="ticker-window"><div className="ticker-track">
    {[0,1].map(d=><div className="ticker-group" key={d} aria-hidden={d===1}>
     {tickerItems.map((t,i)=><span key={i}><Boxes size={13}/>{t}</span>)}
    </div>)}
   </div></div>
  </div>
  <div className="stat-grid">
   <Stat label="Total sparepart" to="/parts" value={data?stats.parts:undefined} detail="Item terdaftar" icon={<Boxes/>}/>
   <Stat label="Perlu restock" to="/parts?low_stock=true" value={data?stats.low_stock:undefined} detail="Stok ≤ batas minimum" icon={<AlertTriangle/>} warn/>
   <Stat label="Service berjalan" to="/services" value={data?stats.active_services:undefined} detail="Job belum selesai" icon={<Wrench/>}/>
   <Stat label="Selesai hari ini" to="/services" value={data?stats.today_completed:undefined} detail="Customer selesai" icon={<CheckCircle2/>}/>
   <Stat label="Omzet hari ini" to="/sales" value={data?stats.today_revenue:undefined} detail="Dari transaksi tercatat" icon={<ReceiptText/>} format={money}/>
  </div>
  <div className="dashboard-grid">
   <section className="panel chart-panel">
    <div className="panel-head"><div><h3>Trend penjualan</h3><p>Omzet harian · 7 hari terakhir</p></div><span className="tag">7 HARI</span></div>
    <div className="chart">{data?<ResponsiveContainer width="100%" height="100%">
     <AreaChart data={data.sales_chart||[]} margin={{top:10,right:12,left:0,bottom:0}}>
      <defs>
       <linearGradient id="gfRevenue" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor="#e07b39" stopOpacity={0.6}/>
        <stop offset="100%" stopColor="#e07b39" stopOpacity={0.03}/>
       </linearGradient>
      </defs>
      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e9edf2"/>
      <XAxis dataKey="day" tickFormatter={v=>v.slice(5)} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:12}}/>
      <YAxis axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:11}} tickFormatter={v=>v>=1000000?`${(v/1000000).toFixed(1)}jt`:v>=1000?`${Math.round(v/1000)}k`:v}/>
      <Tooltip formatter={v=>money(v)} labelFormatter={v=>`Tanggal ${v}`}/>
      <Area type="monotone" dataKey="total" name="Omzet" stroke="#e07b39" strokeWidth={3} fill="url(#gfRevenue)" dot={{r:3,fill:'#e07b39',strokeWidth:0}} activeDot={{r:5}}/>
     </AreaChart>
    </ResponsiveContainer>:<div className="chart-skeleton skeleton"/>}</div>
    <div className="chart-foot"><span><i className="legend-dot"/> Omzet penjualan</span><span>Data transaksi tersimpan</span></div>
   </section>
   <section className="panel">
    <div className="panel-head"><div><h3>Perhatian stok</h3><p>Item yang perlu dicek</p></div><NavLink className="link" to="/parts?low_stock=true">Lihat semua ↗</NavLink></div>
    <div className="stock-list">
     {(data?.low_stock_parts||[]).map(p=><div className="stock-row" key={p.id}>
      <div className="part-icon"><Boxes size={17}/></div>
      <div className="stock-copy"><b>{p.name}</b><small>{p.sku} · {p.location}</small></div>
      <span className={`stock-count ${p.stock===0?'zero':''}`}>{p.stock} <small>pcs</small></span>
     </div>)}
     {!data&&[1,2,3,4].map(i=><div className="stock-row" key={`s${i}`}><div className="part-icon skeleton"/><div className="stock-copy"><span className="skeleton skeleton-line"/></div><span className="skeleton skeleton-pill"/></div>)}
     {data&&!data.low_stock_parts?.length&&<div className="empty"><CheckCircle2/> Semua stok aman</div>}
    </div>
   </section>
  </div>
  <section className="panel chart-panel completed-panel">
   <div className="panel-head"><div><h3>Customer selesai</h3><p>Jumlah service selesai per hari · 7 hari terakhir</p></div><span className="tag">7 HARI</span></div>
   <div className="chart bar-chart">{data?<ResponsiveContainer width="100%" height="100%">
    <BarChart data={data.completed_chart||[]} margin={{top:10,right:12,left:0,bottom:0}}>
     <defs>
      <linearGradient id="gfCompleted" x1="0" y1="0" x2="0" y2="1">
       <stop offset="0%" stopColor="#5fc497"/>
       <stop offset="100%" stopColor="#2f7d5d"/>
      </linearGradient>
     </defs>
     <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e9edf2"/>
     <XAxis dataKey="day" tickFormatter={v=>v.slice(5)} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:12}}/>
     <YAxis allowDecimals={false} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:11}}/>
     <Tooltip formatter={v=>[`${v} customer`,'Selesai']} labelFormatter={v=>`Tanggal ${v}`}/>
     <Bar dataKey="total" name="Selesai" fill="#56a985" radius={[5,5,0,0]} maxBarSize={54}/>
    </BarChart>
   </ResponsiveContainer>:<div className="chart-skeleton skeleton"/>}</div>
   <div className="chart-foot"><span><i className="legend-dot green"/> Customer selesai</span><span>Hari ini: {stats.today_completed??0} customer</span></div>
  </section>
  <section className="panel service-preview">
   <div className="panel-head"><div><h3>Alur service</h3><p>Monitor pekerjaan yang sedang berlangsung</p></div><NavLink className="link" to="/services">Buka service board ↗</NavLink></div>
   <div className="workflow">
    <div><span className="workflow-number">01</span><b>Queued</b><small>Menunggu dikerjakan</small></div>
    <div className="workflow-line"/>
    <div><span className="workflow-number orange">02</span><b>In progress</b><small>Sedang dikerjakan</small></div>
    <div className="workflow-line"/>
    <div><span className="workflow-number green">03</span><b>Completed</b><small>Siap diserahkan</small></div>
   </div>
  </section>
 </>
}

function Stat({label,value,detail,icon,warn,format,to}){
 const nav=useNavigate()
 const click=to?{role:'button',tabIndex:0,onClick:()=>nav(to),onKeyDown:e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();nav(to)}}}:{}
 return <div className={`stat-card ${warn?'is-warning':''} ${to?'clickable':''}`} {...click}>
  <div className="stat-top"><span>{label}</span><span className={`stat-icon ${warn?'warning':''}`}>{icon}</span></div>
  <strong>{value===undefined||value===null?<span className="skeleton skeleton-line"/>:<CountUp value={value} format={format}/>}</strong>
  <small>{detail}</small>
  <span className="led-meter" aria-hidden="true"><i/><i/><i/><i/><i/></span>
 </div>
}

function Parts(){
 const {user}=useAuth()
 const toast=useToast()
 const [items,setItems]=useState([])
 const [movements,setMovements]=useState([])
 const [search,setSearch]=useState('')
 const [lowOnly,setLowOnly]=useState(new URLSearchParams(location.search).get('low_stock')==='true')
 const [show,setShow]=useState(false)
 const [restock,setRestock]=useState(null)
 const [restockForm,setRestockForm]=useState({quantity:1,note:'',moved_at:todayISO()})
 const [selling,setSelling]=useState(null)
 const [sellForm,setSellForm]=useState({quantity:1,sold_at:todayISO()})
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const [form,setForm]=useState({sku:'',name:'',category:'',brand:'',location:'',stock:0,min_stock:3,unit_price:0})
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/parts',{params:{search,low_stock:lowOnly?'true':undefined}}).then(r=>setItems(r.data.data)).catch(()=>setError('Gagal memuat inventory')),
   api.get('/stock-movements').then(r=>setMovements((r.data.data||[]).slice(0,8))).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[search,lowOnly])
 const canManage=user?.role==='manager'
 async function save(e){
  e.preventDefault()
  try{
   const {data}=await api.post('/parts',form)
   setShow(false);setForm({sku:'',name:'',category:'',brand:'',location:'',stock:0,min_stock:3,unit_price:0});load()
   toast(`${data.data.name} ditambahkan ke inventory`,'success')
  }catch(e){setError(e.response?.data?.error||'Gagal menyimpan sparepart')}
 }
 function openSell(p){setSelling(p);setSellForm({quantity:1,sold_at:todayISO()});setError('')}
 async function saveSell(e){
  e.preventDefault()
  try{
   await api.post('/sales',{part_id:selling.id,quantity:Number(sellForm.quantity),sold_at:sellForm.sold_at===todayISO()?'':sellForm.sold_at})
   setSelling(null);load();toast(`Penjualan ${selling.name} tersimpan`,'success')
  }catch(e){setError(e.response?.data?.error||'Transaksi gagal')}
 }
 function openRestock(p){setRestock(p);setRestockForm({quantity:1,note:'',moved_at:todayISO()});setError('')}
 async function saveRestock(e){
  e.preventDefault()
  try{
   await api.post(`/parts/${restock.id}/restock`,{...restockForm,quantity:Number(restockForm.quantity)})
   setRestock(null);load();toast(`Stok ${restock.name} bertambah ${restockForm.quantity} pcs`,'success')
  }catch(e){setError(e.response?.data?.error||'Gagal menambah stok')}
 }
 async function copySku(sku){
  let ok=false
  try{await navigator.clipboard.writeText(sku);ok=true}catch{}
  if(!ok){
   try{
    const ta=document.createElement('textarea');ta.value=sku;ta.setAttribute('readonly','');ta.style.position='fixed';ta.style.opacity='0'
    document.body.appendChild(ta);ta.select();ok=document.execCommand('copy');ta.remove()
   }catch{ok=false}
  }
  ok?toast(`SKU ${sku} disalin ke clipboard`,'info'):toast('Gagal menyalin SKU','error')
 }
 return <>
  <PageTitle eyebrow="STOCK CONTROL" title="Inventory sparepart" no="02" desc="Cari komponen, catat penjualan, dan tambah stok masuk." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/>{canManage&&<button className="primary btn-new" onClick={()=>{setError('');setShow(true)}}><Plus size={17}/> Tambah sparepart</button>}</div>}/>
  {error&&<div className="error">{error}</div>}
  <section className="panel inventory-panel">
   <div className="toolbar">
    <div className="searchbox"><Search size={17}/><input value={search} onChange={e=>setSearch(e.target.value)} placeholder="Cari sparepart, SKU, lokasi…"/></div>
    <label className="check-filter"><input type="checkbox" checked={lowOnly} onChange={e=>setLowOnly(e.target.checked)}/> Stok menipis saja</label>
    <span className="result-count">{items.length} item</span>
   </div>
   <div className="table-wrap"><table className="parts-table">
    <thead><tr><th>SPAREPART</th><th>SKU</th><th>LOKASI GUDANG</th><th>STOK</th><th>HARGA</th><th>STATUS</th><th></th></tr></thead>
    <tbody>{items.map(p=><tr key={p.id}>
     <td><b>{p.name}</b><small className="cell-sub">{p.brand||p.category||'—'}</small></td>
     <td><button type="button" className="sku" title="Klik untuk salin SKU" onClick={()=>copySku(p.sku)}>{p.sku}</button></td>
     <td><span className="location-pill">⌖ {p.location}</span></td>
     <td><b>{p.stock}</b> <small>pcs</small><small className="cell-sub">Min. {p.min_stock}</small></td>
     <td>{money(p.unit_price)}</td>
     <td><span className={`badge ${p.stock<=p.min_stock?'low':'good'}`}>{p.stock===0?'Habis':p.stock<=p.min_stock?'Restock':'Tersedia'}</span></td>
     <td><div className="row-actions">
      <button className="small-action sell" onClick={()=>openSell(p)}><ReceiptText size={12}/> Jual</button>
      {canManage&&<button className="small-action restock" onClick={()=>openRestock(p)}><PackagePlus size={12}/> Restock</button>}
     </div></td>
    </tr>)}</tbody>
   </table>{!items.length&&<div className="empty">Tidak ada sparepart yang cocok.</div>}</div>
  </section>
  <section className="panel inventory-panel stock-history">
   <div className="panel-head"><div><h3>Pergerakan stok terakhir</h3><p>Barang masuk, penjualan, dan pemakaian service</p></div><span className="tag">{movements.length} TERBARU</span></div>
   <div className="table-wrap"><table>
    <thead><tr><th>TANGGAL</th><th>SPAREPART</th><th>TIPE</th><th>JUMLAH</th><th>CATATAN</th></tr></thead>
    <tbody>{movements.map(m=><tr key={m.id}>
     <td>{new Date(m.moved_at).toLocaleDateString('id-ID',{day:'2-digit',month:'short',year:'numeric'})}</td>
     <td><b>{m.part?.name||`Part #${m.part_id}`}</b></td>
     <td><span className={`type-pill ${m.quantity>0?'in':'out'}`}>{movementTypes[m.type]||m.type}</span></td>
     <td className={m.quantity>0?'qty-in':'qty-out'}>{m.quantity>0?'+':''}{m.quantity} pcs</td>
     <td>{m.note||'—'}</td>
    </tr>)}</tbody>
   </table>{!movements.length&&<div className="empty">Belum ada pergerakan stok.</div>}</div>
  </section>
  {show&&<div className="modal-backdrop"><form className="modal" onSubmit={save}>
   <div className="panel-head"><div><h3>Tambah sparepart</h3><p>Lokasi harus spesifik agar mudah ditemukan.</p></div><button type="button" className="icon-btn" onClick={()=>setShow(false)}>×</button></div>
   <div className="form-grid">
    {[['sku','SKU'],['name','Nama sparepart'],['category','Kategori'],['brand','Brand'],['location','Lokasi rak'],['stock','Stok awal'],['min_stock','Stok minimum'],['unit_price','Harga satuan (Rp)']].map(([key,label])=><label key={key}>{label}<input required={!['category','brand'].includes(key)} type={['stock','min_stock','unit_price'].includes(key)?'number':'text'} min={['stock','min_stock','unit_price'].includes(key)?'0':undefined} value={form[key]} onChange={e=>setForm({...form,[key]:e.target.type==='number'?Number(e.target.value):e.target.value})}/></label>)}
   </div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setShow(false)}>Batal</button><button className="primary btn-new">Simpan sparepart</button></div>
  </form></div>}
  {selling&&<div className="modal-backdrop"><form className="modal" onSubmit={saveSell}>
   <div className="panel-head"><div><h3>Catat penjualan</h3><p>{selling.name} · stok tersedia {selling.stock} pcs</p></div><button type="button" className="icon-btn" onClick={()=>setSelling(null)}>×</button></div>
   <div className="sell-summary"><span>Harga satuan</span><b>{money(selling.unit_price)}</b></div>
   <div className="form-grid">
    <label>Jumlah<input type="number" min="1" max={selling.stock} required value={sellForm.quantity} onChange={e=>setSellForm({...sellForm,quantity:e.target.value})}/></label>
    <label>Tanggal penjualan<input type="date" required value={sellForm.sold_at} onChange={e=>setSellForm({...sellForm,sold_at:e.target.value})}/></label>
   </div>
   <div className="picked-total sell-total"><span>Total</span><b>{money(selling.unit_price*Number(sellForm.quantity||0))}</b></div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setSelling(null)}>Batal</button><button className="primary btn-sell"><ReceiptText size={15}/> Simpan penjualan</button></div>
  </form></div>}
  {restock&&<div className="modal-backdrop"><form className="modal" onSubmit={saveRestock}>
   <div className="panel-head"><div><h3>Tambah stok</h3><p>{restock.name} · stok saat ini {restock.stock} pcs</p></div><button type="button" className="icon-btn" onClick={()=>setRestock(null)}>×</button></div>
   <div className="restock-badge"><PackagePlus size={18}/><div><b>{restock.sku}</b><small>{restock.location}</small></div><span>+ stok masuk</span></div>
   <div className="form-grid">
    <label>Jumlah masuk<input type="number" min="1" required value={restockForm.quantity} onChange={e=>setRestockForm({...restockForm,quantity:e.target.value})}/></label>
    <label>Tanggal masuk<input type="date" required value={restockForm.moved_at} onChange={e=>setRestockForm({...restockForm,moved_at:e.target.value})}/></label>
    <label className="full">Catatan<input value={restockForm.note} onChange={e=>setRestockForm({...restockForm,note:e.target.value})} placeholder="Contoh: pembelian dari supplier"/></label>
   </div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setRestock(null)}>Batal</button><button className="primary btn-restock"><PackagePlus size={15}/> Simpan stok</button></div>
  </form></div>}
 </>
}

const statusLabels={queued:'Menunggu',in_progress:'Dikerjakan',waiting_parts:'Menunggu sparepart',completed:'Selesai',cancelled:'Dibatalkan'}
const emptyService={customer_name:'',motorcycle:'',plate_number:'',complaint:'',mechanic_id:0,service_date:todayISO()}

function Services(){
 const toast=useToast()
 const [items,setItems]=useState([])
 const [catalog,setCatalog]=useState([])
 const [show,setShow]=useState(false)
 const [busy,setBusy]=useState(false)
 const [err,setErr]=useState('')
 const [form,setForm]=useState(emptyService)
 const [picked,setPicked]=useState([])
 const [pick,setPick]=useState({part_id:'',quantity:1})
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/services').then(r=>setItems(r.data.data)).catch(()=>setErr('Gagal memuat service')),
   api.get('/parts').then(r=>setCatalog(r.data.data)).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[])
 function openForm(){setForm(emptyService);setPicked([]);setPick({part_id:'',quantity:1});setErr('');setShow(true)}
 function addLine(){
  const id=Number(pick.part_id)
  const qty=Number(pick.quantity)
  const part=catalog.find(x=>x.id===id)
  if(!part||qty<1)return
  const existing=picked.find(x=>x.part_id===id)
  const totalQty=(existing?.quantity||0)+qty
  if(totalQty>part.stock){setErr(`Stok ${part.name} hanya ${part.stock} pcs`);return}
  setErr('')
  if(existing)setPicked(picked.map(x=>x.part_id===id?{...x,quantity:totalQty,subtotal:part.unit_price*totalQty}:x))
  else setPicked([...picked,{part_id:id,quantity:qty,name:part.name,unit_price:part.unit_price,subtotal:part.unit_price*qty}])
  setPick({part_id:'',quantity:1})
 }
 async function create(e){
  e.preventDefault()
  try{
   const {data}=await api.post('/services',{...form,mechanic_id:Number(form.mechanic_id)||0,parts:picked.map(({part_id,quantity})=>({part_id,quantity}))})
   setShow(false);setForm(emptyService);setPicked([]);load()
   toast(`Service ${data.data.work_order} dibuat`,'success')
  }catch(e){setErr(e.response?.data?.error||'Gagal membuat service')}
 }
 async function update(id,status){
  try{
   const {data}=await api.patch(`/services/${id}/status`,{status})
   load()
   toast(status==='completed'?`${data.data.work_order} selesai · sparepart masuk penjualan`:`${data.data.work_order} → ${statusLabels[status]}`,'success')
  }catch(e){setErr(e.response?.data?.error||'Gagal update status')}
 }
 const partsTotal=picked.reduce((a,x)=>a+x.subtotal,0)
 return <>
  <PageTitle eyebrow="WORKSHOP FLOOR" title="Service board" no="03" desc="Pantau kendaraan, keluhan, montir, dan progres pengerjaan." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/><button className="primary btn-new" onClick={openForm}><Wrench size={17}/> Service baru</button></div>}/>
  {err&&<div className="error">{err}</div>}
  <div className="service-columns">
   {['queued','in_progress','waiting_parts','completed'].map(status=><section className="service-column" key={status}>
    <div className="column-heading"><span className={`column-dot ${status}`}/><b>{statusLabels[status]}</b><span className="column-count">{items.filter(x=>x.status===status).length}</span></div>
    {items.filter(x=>x.status===status).map(job=><article className="job-card" key={job.id}>
     <div className="job-top"><span>{job.work_order}</span><Clock3 size={15}/></div>
     <h3>{job.motorcycle}</h3>
     <p className="job-customer">{job.customer_name} · {job.plate_number||'Tanpa plat'}</p>
     <div className="job-complaint">{job.complaint||'Tidak ada catatan keluhan'}</div>
     {(job.parts?.length||0)>0&&<div className="job-parts">{(job.parts||[]).map(p=><span key={p.id}>{p.part?.name||`Part #${p.part_id}`} ×{p.quantity}</span>)}</div>}
     <div className="job-footer"><div className="avatar mini">{job.mechanic?.name?.slice(0,1)||'M'}</div><span>{job.mechanic?.name||'Belum ditugaskan'}</span></div>
     <select value={job.status} onChange={e=>update(job.id,e.target.value)} aria-label="Update status">{Object.entries(statusLabels).map(([v,l])=><option key={v} value={v}>{l}</option>)}</select>
    </article>)}
    {!items.some(x=>x.status===status)&&<div className="column-empty">Belum ada pekerjaan</div>}
   </section>)}
  </div>
  {show&&<div className="modal-backdrop"><form className="modal" onSubmit={create}>
   <div className="panel-head"><div><h3>Buat service order</h3><p>Catat kendaraan, keluhan, dan sparepart yang dibutuhkan.</p></div><button type="button" className="icon-btn" onClick={()=>setShow(false)}>×</button></div>
   <div className="form-grid">
    {[['customer_name','Nama pelanggan'],['motorcycle','Motor'],['plate_number','Nomor polisi'],['complaint','Keluhan']].map(([k,l])=><label key={k}>{l}<input required={k!=='plate_number'&&k!=='complaint'} value={form[k]} onChange={e=>setForm({...form,[k]:e.target.value})}/></label>)}
    <label>Tanggal service<input type="date" required value={form.service_date} onChange={e=>setForm({...form,service_date:e.target.value})}/></label>
   </div>
   <div className="parts-picker">
    <div className="parts-picker-head"><div><b>Sparepart untuk service</b><p>Pilih part yang dipakai. Otomatis tercatat sebagai penjualan saat status Selesai.</p></div><span className="tag">{picked.length} ITEM</span></div>
    <div className="parts-picker-row">
     <select value={pick.part_id} onChange={e=>setPick({...pick,part_id:e.target.value})}>
      <option value="">Pilih sparepart</option>
      {catalog.map(p=><option key={p.id} value={p.id}>{p.name} · stok {p.stock}</option>)}
     </select>
     <input type="number" min="1" value={pick.quantity} onChange={e=>setPick({...pick,quantity:e.target.value})}/>
     <button type="button" className="secondary btn-add" onClick={addLine}><Plus size={14}/> Tambah</button>
    </div>
    {picked.length>0&&<div className="picked-list">
     {picked.map(line=><div className="picked-row" key={line.part_id}>
      <b>{line.name}</b><span>{line.quantity} × {money(line.unit_price)}</span><strong>{money(line.subtotal)}</strong>
      <button type="button" className="icon-btn" onClick={()=>setPicked(picked.filter(x=>x.part_id!==line.part_id))}>×</button>
     </div>)}
     <div className="picked-total"><span>Total sparepart</span><b>{money(partsTotal)}</b></div>
    </div>}
    <div className="hint"><CalendarDays size={14}/> Penjualan sparepart dibuat otomatis ketika job diubah ke status <b>Selesai</b>. Jika dibatalkan, transaksi dibalik dan stok dikembalikan.</div>
   </div>
   {err&&<div className="error">{err}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setShow(false)}>Batal</button><button className="primary btn-new">Buat service</button></div>
  </form></div>}
 </>
}

function Sales(){
 const toast=useToast()
 const [rows,setRows]=useState([])
 const [parts,setParts]=useState([])
 const [part,setPart]=useState('')
 const [qty,setQty]=useState(1)
 const [date,setDate]=useState(todayISO())
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/sales').then(r=>setRows(r.data.data)).catch(()=>setError('Gagal memuat penjualan')),
   api.get('/parts').then(r=>setParts(r.data.data)).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[])
 const selected=parts.find(p=>p.id===Number(part))
 const total=selected?selected.unit_price*Number(qty||0):0
 async function submit(e){
  e.preventDefault()
  try{
   await api.post('/sales',{part_id:Number(part),quantity:Number(qty),sold_at:date===todayISO()?'':date})
   setError('');load();toast('Penjualan tersimpan','success')
  }catch(e){setError(e.response?.data?.error||'Transaksi gagal')}
 }
 return <>
  <PageTitle eyebrow="REVENUE & TRANSACTIONS" title="Penjualan" no="04" desc="Catat penjualan sparepart, termasuk transaksi dari service." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/></div>}/>
  <section className="panel sale-form-panel">
   <div><h3>Transaksi baru</h3><p>Pilih sparepart, jumlah, dan tanggal transaksi.</p></div>
   <form className="sale-form" onSubmit={submit}>
    <select required value={part} onChange={e=>setPart(e.target.value)}><option value="">Pilih sparepart</option>{parts.map(p=><option key={p.id} value={p.id}>{p.name} · stok {p.stock}</option>)}</select>
    <input type="number" min="1" required value={qty} onChange={e=>setQty(e.target.value)}/>
    <input className="date-input" type="date" required value={date} onChange={e=>setDate(e.target.value)} title="Tanggal penjualan"/>
    <button className="primary btn-sell"><ReceiptText size={15}/> Simpan penjualan</button>
   </form>
   {selected&&<div className="sale-preview"><span>Total transaksi</span><b>{money(total)}</b></div>}
   {error&&<div className="error">{error}</div>}
  </section>
  <section className="panel inventory-panel">
   <div className="panel-head"><div><h3>Riwayat transaksi</h3><p>200 transaksi terakhir · manual dan service</p></div><span className="tag">{rows.length} TRANSAKSI</span></div>
   <div className="table-wrap"><table className="sales-table">
    <thead><tr><th>TANGGAL</th><th>SPAREPART</th><th>SUMBER</th><th>JUMLAH</th><th>HARGA SATUAN</th><th>TOTAL</th></tr></thead>
    <tbody>{rows.map(s=><tr key={s.id}>
     <td>{new Date(s.sold_at).toLocaleString('id-ID')}</td>
     <td><b>{s.part?.name||`Part #${s.part_id}`}</b></td>
     <td>{s.source==='service'?<><span className="badge service">Service</span>{s.work_order&&<small className="cell-sub">{s.work_order}</small>}</>:<span className="badge manual">Manual</span>}</td>
     <td>{s.quantity} pcs</td>
     <td>{money(s.unit_price)}</td>
     <td><b>{money(s.total)}</b></td>
    </tr>)}</tbody>
   </table>{!rows.length&&<div className="empty">Belum ada transaksi penjualan.</div>}</div>
  </section>
 </>
}

function Protected(){
 const {user}=useAuth()
 return user?<Shell><Routes><Route path="/" element={<Dashboard/>}/><Route path="/parts" element={<Parts/>}/><Route path="/services" element={<Services/>}/><Route path="/sales" element={<Sales/>}/><Route path="*" element={<Navigate to="/" replace/>}/></Routes></Shell>:<Navigate to="/login" replace/>
}
export default function App(){return <AuthProvider><ToastProvider><Routes><Route path="/login" element={<Login/>}/><Route path="/*" element={<Protected/>}/></Routes></ToastProvider></AuthProvider>}
