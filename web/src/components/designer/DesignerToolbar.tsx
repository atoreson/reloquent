import { Button } from "../Button";

interface DesignerToolbarProps {
  onZoomFit: () => void;
  onAutoLayout: () => void;
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onSave: () => void;
  saving: boolean;
}

export function DesignerToolbar({
  onZoomFit,
  onAutoLayout,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  saving,
}: DesignerToolbarProps) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 shadow-sm">
      <Button variant="secondary" onClick={onZoomFit} className="text-xs px-2 py-1">
        Fit View
      </Button>
      <Button variant="secondary" onClick={onAutoLayout} className="text-xs px-2 py-1">
        Auto Layout
      </Button>
      <div className="w-px h-5 bg-gray-200" />
      <Button
        variant="secondary"
        onClick={onUndo}
        disabled={!canUndo}
        className="text-xs px-2 py-1"
      >
        Undo
      </Button>
      <Button
        variant="secondary"
        onClick={onRedo}
        disabled={!canRedo}
        className="text-xs px-2 py-1"
      >
        Redo
      </Button>
      <div className="flex-1" />
      <Button onClick={onSave} loading={saving} className="text-xs">
        Save & Continue
      </Button>
    </div>
  );
}
