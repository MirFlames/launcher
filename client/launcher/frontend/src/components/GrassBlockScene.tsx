// @ts-nocheck - three.js types incompatible with TS 4.6
import {useRef, useEffect} from 'react';
import * as THREE from 'three';
import {WindowGetPosition} from '../../wailsjs/runtime/runtime';
import './GrassBlockScene.css';

/**
 — v=0 at bottom in Three.js
 * Texture layout 3x4: center=grass top, around=dirt sides, bottom-most=dirt bottom
 */
const ACCENT_COLOR = 0x10b981;

interface GrassBlockSceneProps {
    visible: boolean;
    playHovered?: boolean;
}

export function GrassBlockScene({visible, playHovered = false}: GrassBlockSceneProps) {
    const containerRef = useRef<HTMLDivElement>(null);
    const prevPosRef = useRef<{x: number; y: number} | null>(null);
    const playHoveredRef = useRef(playHovered);
    playHoveredRef.current = playHovered;

    useEffect(() => {
        if (!visible || !containerRef.current) return;

        const width = containerRef.current.clientWidth;
        const height = containerRef.current.clientHeight;
        const scene = new THREE.Scene();
        const camera = new THREE.PerspectiveCamera(50, width / height, 0.1, 100);
        camera.position.z = 4;

        const boxGeo = new THREE.BoxGeometry(1, 1, 1);
        const edgesGeo = new THREE.EdgesGeometry(boxGeo);
        const lineMat = new THREE.LineBasicMaterial({
            color: ACCENT_COLOR,
            transparent: true,
            opacity: 0.5,
        });
        const cube = new THREE.LineSegments(edgesGeo, lineMat);
        scene.add(cube);

        const renderer = new THREE.WebGLRenderer({antialias: true, alpha: true});
        renderer.setClearColor(0x000000, 0);
        renderer.setSize(width, height);
        renderer.setPixelRatio(window.devicePixelRatio);
        containerRef.current.appendChild(renderer.domElement);

        // Physics state
        const pos = {x: 0, y: 0, z: 0};
        const vel = {x: 0, y: 0, z: 0};
        const angVel = {x: 0, y: 0, z: 0};
        const HALF = 0.5;
        const FLOOR_Y = -1.2;
        const GRAVITY = -18;
        const BOUNCE = 0.55;
        const FRICTION = 0.92;
        const ANG_DAMP = 0.98;
        const ANG_VEL_MAX = 6;

        const pollPosition = async () => {
            try {
                const wpos = await WindowGetPosition();
                const prev = prevPosRef.current;
                prevPosRef.current = {x: wpos.x, y: wpos.y};
                if (prev !== null) {
                    const dx = wpos.x - prev.x;
                    const dy = wpos.y - prev.y;
                    const impulse = 0.025;
                    const angImpulse = 0.004;
                    vel.x += dx * impulse;
                    vel.y -= dy * impulse;
                    angVel.y += Math.max(-1, Math.min(1, dx * angImpulse));
                    angVel.x += Math.max(-1, Math.min(1, -dy * angImpulse));
                }
            } catch {
                // ignore
            }
        };

        const pollInterval = setInterval(pollPosition, 50);

        let frameId: number;
        let lastTime = performance.now();
        const animate = () => {
            frameId = requestAnimationFrame(animate);
            const now = performance.now();
            const dt = Math.min((now - lastTime) / 1000, 0.05);
            lastTime = now;

            // Gravity
            vel.y += GRAVITY * dt;

            // Position
            pos.x += vel.x * dt;
            pos.y += vel.y * dt;
            pos.z += vel.z * dt;

            // Floor collision
            if (pos.y - HALF < FLOOR_Y) {
                pos.y = FLOOR_Y + HALF;
                vel.y = -vel.y * BOUNCE;
                vel.x *= FRICTION;
                vel.z *= FRICTION;
                angVel.x *= 0.6;
                angVel.z *= 0.6;
                angVel.y += vel.x * 0.15;
                angVel.z -= vel.z * 0.15;
            }

            // Bounds (invisible walls)
            const BOUND = 1.8;
            if (pos.x - HALF < -BOUND) {
                pos.x = -BOUND + HALF;
                vel.x = -vel.x * BOUNCE;
                angVel.z -= vel.y * 0.08;
            }
            if (pos.x + HALF > BOUND) {
                pos.x = BOUND - HALF;
                vel.x = -vel.x * BOUNCE;
                angVel.z += vel.y * 0.08;
            }
            if (pos.z - HALF < -BOUND) {
                pos.z = -BOUND + HALF;
                vel.z = -vel.z * BOUNCE;
                angVel.x += vel.y * 0.08;
            }
            if (pos.z + HALF > BOUND) {
                pos.z = BOUND - HALF;
                vel.z = -vel.z * BOUNCE;
                angVel.x -= vel.y * 0.08;
            }

            angVel.x = Math.max(-ANG_VEL_MAX, Math.min(ANG_VEL_MAX, angVel.x));
            angVel.y = Math.max(-ANG_VEL_MAX, Math.min(ANG_VEL_MAX, angVel.y));
            angVel.z = Math.max(-ANG_VEL_MAX, Math.min(ANG_VEL_MAX, angVel.z));

            if (playHoveredRef.current) {
                vel.y += 2.5 * dt;
                angVel.x += 0.8 * dt;
                angVel.y += 1.2 * dt;
            }

            angVel.x *= ANG_DAMP;
            angVel.y *= ANG_DAMP;
            angVel.z *= ANG_DAMP;
            cube.rotation.x += angVel.x * dt;
            cube.rotation.y += angVel.y * dt;
            cube.rotation.z += angVel.z * dt;

            cube.position.set(pos.x, pos.y, pos.z);
            renderer.render(scene, camera);
        };
        animate();

        const onResize = () => {
            if (!containerRef.current) return;
            const w = containerRef.current.clientWidth;
            const h = containerRef.current.clientHeight;
            camera.aspect = w / h;
            camera.updateProjectionMatrix();
            renderer.setSize(w, h);
        };
        window.addEventListener('resize', onResize);

        return () => {
            clearInterval(pollInterval);
            cancelAnimationFrame(frameId);
            window.removeEventListener('resize', onResize);
            renderer.dispose();
            lineMat.dispose();
            edgesGeo.dispose();
            boxGeo.dispose();
            if (containerRef.current?.contains(renderer.domElement)) {
                containerRef.current.removeChild(renderer.domElement);
            }
        };
    }, [visible]);

    if (!visible) return null;

    return <div ref={containerRef} className="grass-block-scene" aria-hidden />;
}
