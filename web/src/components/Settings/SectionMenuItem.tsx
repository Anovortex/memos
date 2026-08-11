import { LucideIcon } from "lucide-react";
import React from "react";

interface SectionMenuItemProps {
  text: string;
  icon: LucideIcon;
  isSelected: boolean;
  onClick: () => void;
}

const SectionMenuItem: React.FC<SectionMenuItemProps> = ({ text, icon: IconComponent, isSelected, onClick }) => {
  return (
    <div
      onClick={onClick}
      // Same nav feedback as the icon rail: the active item reads in the accent
      // with a soft shadow, and inactive items resolve toward full contrast on
      // hover instead of dimming.
      className={`w-auto max-w-full px-3 leading-8 flex flex-row justify-start items-center cursor-pointer rounded-lg select-none transition-colors ${
        isSelected ? "bg-accent text-primary shadow-sm" : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
      }`}
    >
      <IconComponent className="w-4 h-auto mr-2 shrink-0" />
      <span className="truncate">{text}</span>
    </div>
  );
};

export default SectionMenuItem;
