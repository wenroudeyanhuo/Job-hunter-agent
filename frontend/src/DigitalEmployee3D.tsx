import { useEffect, useRef } from "react";

type DigitalEmployee3DProps = {
  active: boolean;
  thinking: boolean;
};

export function DigitalEmployee3D({ active, thinking }: DigitalEmployee3DProps) {
  const mountRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) {
      return;
    }
    let disposed = false;
    let cleanup = () => {};
    void import("three").then((THREE) => {
      if (disposed || !mountRef.current) {
        return;
      }
      const scene = new THREE.Scene();
      const camera = new THREE.PerspectiveCamera(35, 1, 0.1, 100);
      camera.position.set(0, 0.15, 5);

      const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      renderer.setSize(180, 220);
      mount.appendChild(renderer.domElement);

      const keyLight = new THREE.DirectionalLight(0xffffff, 2.2);
      keyLight.position.set(2.5, 3, 4);
      scene.add(keyLight);
      scene.add(new THREE.AmbientLight(0x8fb5ff, 1.2));

      const group = new THREE.Group();
      scene.add(group);

      const body = new THREE.Mesh(
        new THREE.CapsuleGeometry(0.72, 1.05, 12, 28),
        new THREE.MeshStandardMaterial({ color: 0x1769e0, roughness: 0.42, metalness: 0.12 }),
      );
      body.position.y = -0.55;
      group.add(body);

      const head = new THREE.Mesh(
        new THREE.SphereGeometry(0.62, 36, 24),
        new THREE.MeshStandardMaterial({ color: 0xf5d4bd, roughness: 0.5 }),
      );
      head.position.y = 0.72;
      group.add(head);

      const visor = new THREE.Mesh(
        new THREE.BoxGeometry(0.78, 0.18, 0.08),
        new THREE.MeshStandardMaterial({ color: 0x18212f, roughness: 0.24, metalness: 0.2 }),
      );
      visor.position.set(0, 0.78, 0.55);
      group.add(visor);

      const halo = new THREE.Mesh(
        new THREE.TorusGeometry(0.82, 0.035, 12, 80),
        new THREE.MeshStandardMaterial({ color: 0x18a36d, emissive: 0x0d7d53, emissiveIntensity: active ? 0.8 : 0.35 }),
      );
      halo.position.y = 1.62;
      halo.rotation.x = Math.PI / 2.8;
      group.add(halo);

      const base = new THREE.Mesh(
        new THREE.CylinderGeometry(0.9, 1.05, 0.18, 42),
        new THREE.MeshStandardMaterial({ color: 0xe8f1ff, roughness: 0.34 }),
      );
      base.position.y = -1.35;
      group.add(base);

      let frame = 0;
      let animationId = 0;
      const animate = () => {
        frame += 0.018;
        group.rotation.y = Math.sin(frame) * 0.18;
        group.position.y = Math.sin(frame * 1.7) * 0.04;
        halo.rotation.z += thinking ? 0.035 : 0.012;
        const pulse = active || thinking ? 0.08 : 0.03;
        halo.scale.setScalar(1 + Math.sin(frame * 4) * pulse);
        renderer.render(scene, camera);
        animationId = requestAnimationFrame(animate);
      };
      animate();

      const resizeObserver = new ResizeObserver((entries) => {
        const rect = entries[0]?.contentRect;
        if (!rect) {
          return;
        }
        const width = Math.max(150, Math.floor(rect.width));
        const height = Math.max(180, Math.floor(rect.height));
        renderer.setSize(width, height);
        camera.aspect = width / height;
        camera.updateProjectionMatrix();
      });
      resizeObserver.observe(mount);

      cleanup = () => {
        cancelAnimationFrame(animationId);
        resizeObserver.disconnect();
        if (renderer.domElement.parentElement === mount) {
          mount.removeChild(renderer.domElement);
        }
        renderer.dispose();
        scene.traverse((object) => {
          if (object instanceof THREE.Mesh) {
            object.geometry.dispose();
            const material = object.material;
            if (Array.isArray(material)) {
              material.forEach((item) => item.dispose());
            } else {
              material.dispose();
            }
          }
        });
      };
    });

    return () => {
      disposed = true;
      cleanup();
    };
  }, [active, thinking]);

  return <div className="employee-3d-stage" ref={mountRef} aria-label="Digital employee 3D avatar" />;
}
