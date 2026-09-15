import { Routes, Route } from 'react-router-dom'
import textureBand from './assets/texture-band.png'
import poster from './assets/card-bright.png'
import InventoryPage from './pages/Inventory'
import IntakePage from './pages/Intake'
import ItemPage from './pages/Item'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Splash />} />
      <Route path="/inventory" element={<InventoryPage />} />
      <Route path="/inventory/new" element={<IntakePage />} />
      <Route path="/inventory/item/:id" element={<ItemPage />} />
    </Routes>
  )
}

// Splash is the public_site landing page, kept hardcoded as before.
function Splash() {
  return (
    <div className="flex min-h-screen flex-col bg-[#2e2418]">
      <div
        className="h-[70px] bg-[length:auto_100%] bg-repeat-x"
        style={{ backgroundImage: `url(${textureBand})` }}
      />
      <main className="flex grow flex-col items-center justify-center px-6 py-12 text-center">
        <img
          src={poster}
          alt="Up 'n' Bright vintage poster art"
          className="h-auto w-[min(300px,80vw)] -rotate-2 rounded-[10px] shadow-[0_12px_40px_rgba(0,0,0,0.6)]"
        />
        <h1 className="mt-9 mb-0 font-serif text-[2.2rem] font-bold text-[#e8d9b8]">
          Bright Vintage Finds
        </h1>
        <p className="mt-3 mb-0 text-[0.85rem] tracking-[0.14em] text-[#a3927c] uppercase">
          Full site coming soon
        </p>
      </main>
      <div
        className="h-[70px] -scale-y-100 bg-[length:auto_100%] bg-repeat-x"
        style={{ backgroundImage: `url(${textureBand})` }}
      />
    </div>
  )
}
