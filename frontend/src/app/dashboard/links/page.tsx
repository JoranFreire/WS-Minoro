"use client";

import { useState } from "react";
import { useLinks } from "@/hooks/useLinks";
import { LinkCard } from "@/components/links/LinkCard";
import { LinkForm } from "@/components/links/LinkForm";
import { Link } from "@/lib/api";
import { Plus } from "lucide-react";

export default function LinksPage() {
  const { data: links, isLoading, error } = useLinks();
  const [showForm, setShowForm] = useState(false);
  const [editingLink, setEditingLink] = useState<Link | undefined>();

  const handleEdit = (link: Link) => {
    setEditingLink(link);
    setShowForm(true);
  };

  const handleClose = () => {
    setShowForm(false);
    setEditingLink(undefined);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Links</h1>
          <p className="text-gray-500 text-sm mt-0.5">
            {links?.length ?? 0} link{links?.length !== 1 ? "s" : ""}
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          New link
        </button>
      </div>

      {showForm && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              {editingLink ? "Edit link" : "Create link"}
            </h2>
            <LinkForm link={editingLink} onClose={handleClose} />
          </div>
        </div>
      )}

      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-5 h-36 animate-pulse">
              <div className="h-4 bg-gray-100 rounded w-3/4 mb-2" />
              <div className="h-3 bg-gray-100 rounded w-1/2" />
            </div>
          ))}
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          Failed to load links. Make sure the API is running.
        </div>
      )}

      {!isLoading && links && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {links.map((link) => (
            <LinkCard key={link.id} link={link} onEdit={handleEdit} />
          ))}
          {links.length === 0 && (
            <div className="col-span-3 text-center py-16 text-gray-400">
              <p className="text-lg font-medium">No links yet</p>
              <p className="text-sm mt-1">Create your first link to get started</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
