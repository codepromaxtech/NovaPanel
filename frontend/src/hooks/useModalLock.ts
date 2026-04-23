import { useEffect } from 'react';

/** Locks body scroll while a modal is open. Pass `true` when modal is visible. */
export function useModalLock(isOpen: boolean) {
    useEffect(() => {
        if (!isOpen) return;
        const prev = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        return () => {
            document.body.style.overflow = prev;
        };
    }, [isOpen]);
}
