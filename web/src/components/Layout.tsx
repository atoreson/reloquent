import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";

export function Layout() {
  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <main className="flex-1 overflow-y-auto p-8">
        <div className="mx-auto max-w-4xl">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
