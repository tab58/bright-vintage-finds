import { Routes, Route } from 'react-router-dom';
import HomePage from './pages/Home';
import InventoryPage from './pages/Inventory';
import IntakePage from './pages/Intake';
import ItemPage from './pages/Item';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/inventory" element={<InventoryPage />} />
      <Route path="/inventory/new" element={<IntakePage />} />
      <Route path="/inventory/item/:id" element={<ItemPage />} />
    </Routes>
  );
}
