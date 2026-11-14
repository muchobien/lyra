import "react";

declare module "react" {
  interface ButtonHTMLAttributes<T> {
    commandfor?: string;
    command?: "show-modal" | "close" | "request-close";
  }
  interface DialogHTMLAttributes<T> {
    closedby?: "any" | "closerequest" | "none";
  }
}
