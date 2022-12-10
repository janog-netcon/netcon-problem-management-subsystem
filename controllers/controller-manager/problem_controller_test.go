package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Problem controller", func() {
	ctx := context.Background()

	var stopFunc func()

	BeforeEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &netconv1alpha1.Worker{})
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &netconv1alpha1.Problem{}, client.InNamespace("default"))
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &netconv1alpha1.ProblemEnvironment{}, client.InNamespace("default"))
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme.Scheme,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&ProblemReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should meet assignableReplicas when assignableReplicas is updated", func() {
		problem := netconv1alpha1.Problem{}
		err := loadManifest(filepath.Join("tests", "problems", "problem-tst-001.yaml"), &problem)
		Expect(err).NotTo(HaveOccurred())

		problemNamespace := problem.Namespace
		problemName := problem.Name

		err = k8sClient.Create(ctx, &problem)
		Expect(err).NotTo(HaveOccurred())

		updateAssignableReplicas := func(assignableReplicas int) {
			problem := netconv1alpha1.Problem{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: problemNamespace,
				Name:      problemName,
			}, &problem)
			Expect(err).NotTo(HaveOccurred())

			problem.Spec.AssignableReplicas = assignableReplicas
			err = k8sClient.Update(ctx, &problem)
			Expect(err).NotTo(HaveOccurred())
		}

		checkAssignableReplias := func(assignableReplicas int) AsyncAssertion {
			return Eventually(func() error {
				problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
				if err := k8sClient.List(ctx, &problemEnvironments); err != nil {
					return err
				}

				if len(problemEnvironments.Items) != assignableReplicas {
					return fmt.Errorf("got %d items", len(problemEnvironments.Items))
				}

				return nil
			})
		}

		// first, assignableReplicas == 1
		checkAssignableReplias(1).Should(Succeed())

		// next, update assignableReplicas to 5
		updateAssignableReplicas(5)
		checkAssignableReplias(5).Should(Succeed())

		// and then, update assignableReplicas to 0
		updateAssignableReplicas(0)
		checkAssignableReplias(0).Should(Succeed())

		// finally, update assignableReplicas to 3
		updateAssignableReplicas(3)
		checkAssignableReplias(3).Should(Succeed())
	})
})
