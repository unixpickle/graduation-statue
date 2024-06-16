package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/model3d/render3d"
	"github.com/unixpickle/model3d/toolbox3d"
)

const Resolution = 0.0075

func main() {
	computer, _ := LoadAsset("computer")
	baseContainer, _ := LoadAsset("base")
	screenText := ScreenText()
	screenMesh := ScreenTris()
	screenContainer := model3d.NewColliderSolidHollow(model3d.MeshToCollider(screenMesh), 0.01)

	screenTextSolid := model3d.TranslateSolid(
		model3d.IntersectedSolid{computer, screenText},
		model3d.Y(-0.02),
	)

	var computerColor toolbox3d.CoordColorFunc = func(c model3d.Coord3D) render3d.Color {
		if screenText.Contains(c) {
			return render3d.NewColor(1.0)
		} else if screenContainer.Contains(c) {
			return render3d.NewColor(0)
		} else if baseContainer.Contains(c) {
			return render3d.NewColorRGB(0xC5/255.0, 0x76/255.0, 0xF6/255.0)
		} else {
			return render3d.NewColor(1)
		}
	}
	cap, capColor := GraduationCap()
	mesh, interior := model3d.DualContourInterior(
		model3d.JoinedSolid{computer, cap, screenTextSolid}, Resolution, true, false,
	)

	fmt.Println("size:", mesh.Max().Sub(mesh.Min()))

	colorFunc := toolbox3d.JoinedSolidCoordColorFunc(
		interior,
		cap, capColor,
		computer, computerColor,
		screenTextSolid, toolbox3d.ConstantCoordColorFunc(render3d.NewColor(1.0)),
	)
	render3d.SaveRandomGrid("rendering.png", mesh, 3, 3, 300, colorFunc.RenderColor)
	mesh.SaveMaterialOBJ("computer.zip", colorFunc.TriangleColor)
}

func GraduationCap() (model3d.Solid, toolbox3d.CoordColorFunc) {
	base, _ := LoadAsset("hat_base")
	bottom := model3d.IntersectedSolid{
		model3d.NewRect(base.Min().Add(model3d.Z(-0.05)), base.Max().Add(model3d.Z(0.3))),
		&model3d.Cone{
			Tip:    base.Max().Mid(base.Min()).Add(model3d.Z(3.0)),
			Base:   base.Max().Mid(base.Min()).Add(model3d.Z(-3.0)),
			Radius: 0.8,
		},
	}
	top := model3d.TranslateSolid(base, model3d.Z(0.3))
	start := top.Min().Mid(top.Max())
	width := top.Max().X - start.X

	tassleCurve := model2d.JoinedCurve{
		model2d.BezierCurve{
			model2d.XY(start.X, start.Z),
			model2d.XY(start.X, start.Z+0.075),
			model2d.XY(start.X+0.075, start.Z+0.075),
		},
		model2d.BezierCurve{
			model2d.XY(start.X+0.075, start.Z+0.075),
			model2d.XY(start.X+width-0.075, start.Z+0.075),
		},
		model2d.BezierCurve{
			model2d.XY(start.X+width-0.075, start.Z+0.075),
			model2d.XY(start.X+width, start.Z+0.075),
			model2d.XY(start.X+width, start.Z),
		},
		model2d.BezierCurve{
			model2d.XY(start.X+width, start.Z),
			model2d.XY(start.X+width, start.Z-0.2),
		},
	}
	tasslePoint := func(t float64) model3d.Coord3D {
		p := tassleCurve.Eval(t)
		return model3d.XYZ(p.X, start.Y, p.Y)
	}
	lines := []model3d.Segment{}
	eps := 0.01
	for t := eps; t <= 1.0; t += eps {
		lines = append(lines, model3d.NewSegment(tasslePoint(t-eps), tasslePoint(t)))
	}
	tassle := toolbox3d.LineJoin(0.05, lines...)
	fullTassle := model3d.JoinedSolid{
		tassle,
		&model3d.Sphere{
			Center: model3d.XYZ(start.X+width, start.Y, start.Z-0.15),
			Radius: 0.08,
		},
		&model3d.Cone{
			Tip:    tasslePoint(0.85),
			Base:   model3d.XYZ(start.X+width, start.Y, start.Z-0.4),
			Radius: 0.13,
		},
	}

	return model3d.JoinedSolid{bottom, top, fullTassle}, func(c model3d.Coord3D) render3d.Color {
		if fullTassle.Contains(c) {
			return render3d.NewColorRGB(0xFF/255.0, 0xD7/255.0, 0.0)
		}
		return render3d.NewColor(0.0)
	}
}

func LoadAsset(name string) (model3d.Solid, *model3d.Mesh) {
	f, err := os.Open(filepath.Join("assets", name+".stl"))
	essentials.Must(err)
	defer f.Close()
	tris, err := model3d.ReadSTL(f)
	essentials.Must(err)
	mesh := model3d.NewMeshTriangles(tris)
	return model3d.NewColliderSolid(model3d.MeshToCollider(mesh)), mesh
}

func ScreenText() model3d.Solid {
	screen, _ := LoadAsset("screen")
	screenMid := screen.Max().Mid(screen.Min())

	mesh2d := model2d.MustReadBitmap("assets/text.png", nil).FlipY().Mesh().SmoothSq(20)
	scale := 0.8 * (screen.Max().X - screen.Min().X) / (mesh2d.Max().X - mesh2d.Min().X)
	mesh2d = mesh2d.Scale(scale)
	mesh2d = mesh2d.Translate(mesh2d.Min().Mid(mesh2d.Max()).Scale(-1))
	mesh2d = mesh2d.Translate(model2d.XY(screenMid.X, screenMid.Z))
	solid2d := model2d.NewColliderSolid(model2d.MeshToCollider(mesh2d))

	return model3d.CheckedFuncSolid(screen.Min(), screen.Max(), func(c model3d.Coord3D) bool {
		return solid2d.Contains(c.XZ())
	})
}

func ScreenTris() *model3d.Mesh {
	_, computer := LoadAsset("computer")
	screen, _ := LoadAsset("screen")
	center := screen.Min().Mid(screen.Max())
	center.Y = screen.Min().Y
	collision, collides := model3d.MeshToCollider(computer).FirstRayCollision(&model3d.Ray{
		Origin:    center,
		Direction: model3d.Y(1),
	})
	if !collides {
		panic("could not find a screen triangle")
	}
	tri := collision.Extra.(*model3d.TriangleCollision).Triangle
	queue := []*model3d.Triangle{tri}
	result := model3d.NewMesh()
	computer.Remove(tri)
	result.Add(tri)
	for len(queue) > 0 {
		obj := queue[0]
		queue = queue[1:]
		for _, neighbor := range computer.Neighbors(obj) {
			normal := neighbor.Normal()
			if math.Abs(normal.Y) < 0.8 || neighbor.Normal().Dot(obj.Normal()) < 0.995 {
				continue
			}
			result.Add(neighbor)
			computer.Remove(neighbor)
			queue = append(queue, neighbor)
		}
	}
	return result
}
