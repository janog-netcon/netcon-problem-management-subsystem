package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	util "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		time.Sleep(100 * time.Millisecond)

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
		time.Sleep(100 * time.Millisecond)

		// next, update assignableReplicas to 5
		updateAssignableReplicas(5)
		checkAssignableReplias(5).Should(Succeed())
		time.Sleep(100 * time.Millisecond)

		// and then, update assignableReplicas to 0
		updateAssignableReplicas(0)
		checkAssignableReplias(0).Should(Succeed())
		time.Sleep(100 * time.Millisecond)

		// finally, update assignableReplicas to 3
		updateAssignableReplicas(3)
		checkAssignableReplias(3).Should(Succeed())
		time.Sleep(100 * time.Millisecond)
	})

	It("should meet assignableReplicas when ProblemEnvironment is assigned", func() {
		problem := netconv1alpha1.Problem{}
		err := loadManifest(filepath.Join("tests", "problems", "problem-tst-002.yaml"), &problem)
		Expect(err).NotTo(HaveOccurred())

		problemNamespace := problem.Namespace
		problemName := problem.Name

		err = k8sClient.Create(ctx, &problem)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}

		updateAssignableReplicas := func(assignableReplicas int) {
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: problemNamespace,
				Name:      problemName,
			}, &problem)
			Expect(err).NotTo(HaveOccurred())

			problem.Spec.AssignableReplicas = assignableReplicas
			err = k8sClient.Update(ctx, &problem)
			Expect(err).NotTo(HaveOccurred())
		}

		checkReplicas := func(assignableReplicas, desiredAssignedReplicas int) AsyncAssertion {
			return Eventually(func() error {
				if err := k8sClient.List(ctx, &problemEnvironments); err != nil {
					return err
				}

				assigned, notAssigned := 0, 0

				for _, problemEnvironment := range problemEnvironments.Items {
					if util.GetProblemEnvironmentCondition(
						&problemEnvironment,
						netconv1alpha1.ProblemEnvironmentConditionAssigned,
					) == metav1.ConditionTrue {
						assigned += 1
					} else {
						notAssigned += 1
					}
				}

				if notAssigned != assignableReplicas || assigned != desiredAssignedReplicas {
					return fmt.Errorf("%d assignable, %d assigned", notAssigned, assigned)
				}

				return nil
			})
		}

		// first, 3 assignableReplicas are expected to exist
		checkReplicas(3, 0).Should(Succeed())
		time.Sleep(100 * time.Millisecond)

		// set all assignableReplicas as assigned
		for _, problemEnvironment := range problemEnvironments.Items {
			util.SetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
				metav1.ConditionTrue,
				"TEST", "---")
			util.SetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
				metav1.ConditionTrue,
				"TEST", "---")
			util.SetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionAssigned,
				metav1.ConditionTrue,
				"TEST", "---")

			err = k8sClient.Status().Update(ctx, &problemEnvironment)
			Expect(err).NotTo(HaveOccurred())
		}
		time.Sleep(100 * time.Millisecond)

		// now, all assignableReplicas were assigned
		// so, 3 assignableReplicas and 3 assignedReplicas are expected to exist
		checkReplicas(3, 3).Should(Succeed())
		time.Sleep(100 * time.Millisecond)

		// next, update assignableReplicas to 1
		// so, 1 assignableReplicas and 3 assignedReplicas are expected to exist
		// This ensures even if assignedReplicas is decreased, assigned replicas will never deleted
		updateAssignableReplicas(1)
		checkReplicas(1, 3).Should(Succeed())
		time.Sleep(100 * time.Millisecond)
	})
})
