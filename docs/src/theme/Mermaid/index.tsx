import React, {useState, useRef, useCallback, useEffect, type ReactNode} from 'react';
import ErrorBoundary from '@docusaurus/ErrorBoundary';
import {ErrorBoundaryErrorMessageFallback} from '@docusaurus/theme-common';
import {
  MermaidContainerClassName,
  useMermaidRenderResult,
} from '@docusaurus/theme-mermaid/client';
import type {Props} from '@theme/Mermaid';
import type {RenderResult} from 'mermaid';

import styles from './styles.module.css';

function MermaidSvg({
  renderResult,
  containerClassName,
}: {
  renderResult: RenderResult;
  containerClassName?: string;
}): ReactNode {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const div = ref.current!;
    renderResult.bindFunctions?.(div);
  }, [renderResult]);

  return (
    <div
      ref={ref}
      className={`${MermaidContainerClassName} ${styles.svgContainer} ${containerClassName ?? ''}`}
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{__html: renderResult.svg}}
    />
  );
}

const clamp = (value: number, min: number, max: number): number =>
  Math.min(Math.max(value, min), max);

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 8;

function ExpandableDiagram({renderResult}: {renderResult: RenderResult}): ReactNode {
  const [expanded, setExpanded] = useState(false);

  const close = useCallback(() => setExpanded(false), []);
  useEffect(() => {
    if (!expanded) {
      return undefined;
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setExpanded(false);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [expanded]);

  return (
    <>
      <div className={styles.wrapper}>
        <button
          type="button"
          className={styles.expandButton}
          aria-label="Expand diagram"
          title="Expand diagram"
          onClick={() => setExpanded(true)}>
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path d="M15 3h6v6h-2V5h-4V3zM9 3v2H5v4H3V3h6zm6 18h6v-6h-2v4h-4v2zm-6 0v-2H5v-4H3v6h6z" />
          </svg>
        </button>
        <div
          className={styles.inline}
          role="button"
          tabIndex={0}
          aria-label="Expand diagram"
          onClick={() => setExpanded(true)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              setExpanded(true);
            }
          }}>
          <MermaidSvg renderResult={renderResult} />
        </div>
      </div>

      {expanded ? (
        <Overlay renderResult={renderResult} onClose={close} />
      ) : null}
    </>
  );
}

function Overlay({
  renderResult,
  onClose,
}: {
  renderResult: RenderResult;
  onClose: () => void;
}): ReactNode {
  const stageRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{
    pointerId: number;
    startX: number;
    startY: number;
    originX: number;
    originY: number;
  } | null>(null);

  const [transform, setTransform] = useState({x: 0, y: 0, zoom: 1});

  const onWheel = useCallback(
    (event: React.WheelEvent<HTMLDivElement>) => {
      const view = viewRef.current;
      if (!view) {
        return;
      }
      event.preventDefault();
      const rect = view.getBoundingClientRect();
      const scale = Math.exp(-event.deltaY * 0.0016);
      setTransform((prev) => {
        const zoom = clamp(prev.zoom * scale, MIN_ZOOM, MAX_ZOOM);
        const px = event.clientX - rect.left;
        const py = event.clientY - rect.top;
        return {
          zoom,
          x: px - (px - prev.x) * (zoom / prev.zoom),
          y: py - (py - prev.y) * (zoom / prev.zoom),
        };
      });
    },
    [],
  );

  const onPointerDown = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    const stage = stageRef.current;
    if (!stage) {
      return;
    }
    stage.setPointerCapture(event.pointerId);
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      originX: transform.x,
      originY: transform.y,
    };
  }, [transform]);

  const onPointerMove = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) {
      return;
    }
    setTransform((prev) => ({
      ...prev,
      x: drag.originX + (event.clientX - drag.startX),
      y: drag.originY + (event.clientY - drag.startY),
    }));
  }, []);

  const endDrag = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId === event.pointerId) {
      dragRef.current = null;
    }
  }, []);

  return (
    <div className={styles.backdrop} onWheel={onWheel}>
      <button
        type="button"
        className={styles.closeButton}
        aria-label="Close and exit expanded mode"
        onClick={onClose}>
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
          <path d="M18.3 5.7 12 12l6.3 6.3-1.4 1.4L10.6 13.4 4.3 19.7 2.9 18.3 9.2 12 2.9 5.7 4.3 4.3 10.6 10.6 16.9 4.3z" />
        </svg>
      </button>
      <div className={styles.toolbar}>
        <button
          type="button"
          className={styles.toolButton}
          onClick={() => setTransform((p) => ({...p, zoom: clamp((p.zoom + 0.25) * 1.2, MIN_ZOOM, MAX_ZOOM)}))}>
          +
        </button>
        <button
          type="button"
          className={styles.toolButton}
          onClick={() => setTransform((p) => ({...p, zoom: clamp(p.zoom / 1.2, MIN_ZOOM, MAX_ZOOM)}))}>
          −
        </button>
        <button
          type="button"
          className={styles.toolButton}
          onClick={() => setTransform({x: 0, y: 0, zoom: 1})}>
          Reset
        </button>
      </div>
      <div className={styles.viewport} ref={viewRef}>
        <div
          ref={stageRef}
          className={styles.stage}
          role="application"
          aria-label="Pan and zoom the diagram. Drag to pan, scroll to zoom."
          style={{
            transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.zoom})`,
          }}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}>
          <MermaidSvg renderResult={renderResult} containerClassName={styles.large} />
        </div>
      </div>
    </div>
  );
}

function MermaidRenderer({value}: Props): ReactNode {
  const renderResult = useMermaidRenderResult({text: value});
  if (renderResult === null) {
    return null;
  }
  return <ExpandableDiagram renderResult={renderResult} />;
}

export default function Mermaid(props: Props): ReactNode {
  return (
    <ErrorBoundary
      fallback={(params) => <ErrorBoundaryErrorMessageFallback {...params} />}>
      <MermaidRenderer {...props} />
    </ErrorBoundary>
  );
}
